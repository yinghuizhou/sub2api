import { chromium } from 'playwright';
import type { Browser, Page } from 'playwright';
import https from 'https';
import http from 'http';
import { mkdirSync, writeFileSync, readdirSync } from 'fs';
import { join, basename } from 'path';
import { getState, setState, setMsg, broadcast } from './state';
import { loadConfig, saveConfig, loadCookies, saveCookies, OVPN_DIR } from './config';

const ASTRILL_CERTS = 'https://www.astrill.com/member-zone/tools/openvpn-certificates';
const MAX_RETRY = 3;

// Persist browser/page across HMR reloads
const g = globalThis as Record<string, unknown>;
if (!g.__astrill_runtime) {
  g.__astrill_runtime = {
    browser: null as Browser | null,
    page: null as Page | null,
    stopRequested: false,
    captchaResolve: null as ((v: string | null) => void) | null,
    cookieStr: '',
  };
}

interface Runtime {
  browser: Browser | null;
  page: Page | null;
  stopRequested: boolean;
  captchaResolve: ((v: string | null) => void) | null;
  cookieStr: string;
}

const rt = g.__astrill_runtime as Runtime;

export function getPage(): Page | null { return rt.page; }

export function requestStop(): void {
  rt.stopRequested = true;
  if (rt.captchaResolve) rt.captchaResolve(null);
  rt.page?.evaluate(() => window.stop()).catch(() => { /* ignore */ });
}

export function resolveCaptcha(solution: string | null): void {
  if (rt.captchaResolve) rt.captchaResolve(solution);
}

function stopRace(signal: AbortSignal): Promise<never> {
  return new Promise((_, reject) => {
    const t = setInterval(() => {
      if (rt.stopRequested) { clearInterval(t); reject(new Error('STOPPED')); }
    }, 200);
    signal.addEventListener('abort', () => clearInterval(t), { once: true });
  });
}

function waitForCaptcha(): Promise<string | null> {
  // Resolve any previously pending captcha to avoid Promise leak on rapid re-calls
  if (rt.captchaResolve) rt.captchaResolve(null);
  return new Promise(r => { rt.captchaResolve = r; });
}

function setupCertIdInterception(): void {
  const page = rt.page!;
  page.on('request', req => {
    const m = req.url().match(/\/openvpn-certificates\/download\/(\d{5,12})\//);
    if (m && !getState().certId) {
      setState({ certId: m[1] });
      console.log(`certId captured: ${m[1]}`);
    }
  });
  page.on('response', async res => {
    if (getState().certId) return;
    const url = res.url();
    const m = url.match(/\/openvpn-certificates\/download\/(\d{5,12})\//);
    if (m) { setState({ certId: m[1] }); return; }
    if (url.includes('astrill.com') && res.headers()['content-type']?.includes('json')) {
      try {
        const text = await res.text().catch(() => '');
        const jm = text.match(/["']?cert[_-]?id["']?\s*[:=]\s*["']?(\d{5,12})["']?/i);
        if (jm) setState({ certId: jm[1] });
      } catch { /* ignore */ }
    }
  });
}

async function closeModal(): Promise<void> {
  const page = rt.page!;
  try {
    await page.getByText('关闭窗口').click().catch(() => { /* ignore */ });
    await page.waitForSelector('.modal.show, .modal[style*="display: block"], .modal.in', {
      state: 'hidden', timeout: 3000
    }).catch(() => { /* ignore */ });
    await page.waitForTimeout(300);
  } catch { /* ignore */ }
}

async function extractCertIdFromPage(): Promise<string | null> {
  return rt.page!.evaluate(() => {
    for (const el of document.querySelectorAll('[onclick]')) {
      const m = (el.getAttribute('onclick') || '').match(/(\d{5,12})/);
      if (m && el.getAttribute('onclick')!.includes('Download')) return m[1];
    }
    for (const s of document.querySelectorAll('script:not([src])')) {
      const t = s.textContent || '';
      const m = t.match(/OnDownloadSingle\(['"](\d+)['"]\)/);
      if (m) return m[1];
      const m2 = t.match(/cert[_-]?id['":\s]*[=:]\s*['"]?(\d{5,12})['"]?/i);
      if (m2) return m2[1];
    }
    for (const el of document.querySelectorAll('[data-cert-id],[data-certid],[data-certificate-id]')) {
      const v = el.getAttribute('data-cert-id') || el.getAttribute('data-certid') || el.getAttribute('data-certificate-id');
      if (v && /^\d{5,12}$/.test(v)) return v;
    }
    for (const key of ['certId', 'cert_id', 'CERT_ID', 'certificateId']) {
      const w = window as unknown as Record<string, unknown>;
      if (w[key] && /^\d{5,12}$/.test(String(w[key]))) return String(w[key]);
    }
    const inp = document.querySelector<HTMLInputElement>('input[name="cert_id"],input[name="certId"]');
    if (inp?.value && /^\d{5,12}$/.test(inp.value)) return inp.value;
    const m = document.documentElement.innerHTML.match(/\/download\/(\d{5,12})\//);
    if (m) return m[1];
    const m2 = document.documentElement.innerHTML.match(/OnDownloadSingle\(['"](\d+)['"]\)/);
    if (m2) return m2[1];
    return null;
  });
}

async function extractCertId(): Promise<string | null> {
  if (getState().certId) return getState().certId;
  let certId = await extractCertIdFromPage();
  if (!certId) {
    const cfg = loadConfig();
    if (cfg.lastCertId) certId = cfg.lastCertId;
  }
  return certId;
}

function httpsRequest(url: string, opts: {
  method?: string; headers?: Record<string, string>; body?: string; timeout?: number;
} = {}): Promise<{ status: number; headers: Record<string, string>; body: Buffer }> {
  const isHttps = url.startsWith('https');
  const mod = isHttps ? https : http;
  return new Promise((resolve, reject) => {
    const u = new URL(url);
    const bodyBuf = opts.body ? Buffer.from(opts.body) : null;
    const headers: Record<string, string> = { 'User-Agent': 'Mozilla/5.0', ...opts.headers };
    if (bodyBuf) headers['Content-Length'] = String(bodyBuf.length);
    const req = mod.request({
      hostname: u.hostname, port: u.port ? Number(u.port) : (isHttps ? 443 : 80),
      path: u.pathname + u.search, method: opts.method || 'GET',
      headers, timeout: opts.timeout || 30000,
    } as import('https').RequestOptions, res => {
      const chunks: Buffer[] = [];
      res.on('data', c => chunks.push(c as Buffer));
      res.on('end', () => resolve({
        status: res.statusCode || 0,
        headers: res.headers as Record<string, string>,
        body: Buffer.concat(chunks),
      }));
    });
    req.on('timeout', () => { req.destroy(); reject(new Error(`Timeout: ${url}`)); });
    req.on('error', reject);
    if (bodyBuf) req.write(bodyBuf);
    req.end();
  });
}

async function recognizeCaptcha(imgBase64: string): Promise<string> {
  const cfg = loadConfig();
  if (!cfg.apiUrl || !cfg.apiKey) { setMsg('⚠️ 未配置 AI 接口，跳过自动识别', 'warn'); return ''; }
  try {
    const endpoint = cfg.apiUrl.replace(/\/+$/, '') + '/v1/messages';
    const body = JSON.stringify({
      model: cfg.model || 'claude-haiku-4-5-20251001', max_tokens: 50,
      messages: [{ role: 'user', content: [
        { type: 'image', source: { type: 'base64', media_type: 'image/jpeg', data: imgBase64 } },
        { type: 'text', text: 'What letters/characters appear in this image? Output only those exact characters, preserving case. No explanation.' },
      ]}],
    });
    const res = await httpsRequest(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'x-api-key': cfg.apiKey, 'anthropic-version': '2023-06-01' },
      body, timeout: 15000,
    });
    const data = JSON.parse(res.body.toString()) as { error?: { message: string }; content?: Array<{ text: string }> };
    if (data.error) { setMsg(`⚠️ AI 识别失败: ${data.error.message}`, 'warn'); return ''; }
    if (data.content?.[0]?.text) return data.content[0].text.trim();
  } catch (e) { setMsg(`⚠️ AI 识别出错: ${(e as Error).message}`, 'warn'); }
  return '';
}

async function ensureLoggedIn(): Promise<void> {
  if (rt.stopRequested) throw new Error('STOPPED');
  const page = rt.page!;
  const gotoCtrl = new AbortController();
  await Promise.race([
    page.goto(ASTRILL_CERTS, { waitUntil: 'domcontentloaded', timeout: 30000 }),
    stopRace(gotoCtrl.signal),
  ]);
  gotoCtrl.abort();
  const idleCtrl = new AbortController();
  await Promise.race([
    page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => { /* ignore */ }),
    stopRace(idleCtrl.signal),
  ]);
  idleCtrl.abort();
  const url = page.url();
  if (url.includes('log-in')) {
    setMsg('🔑 会话已过期，请在浏览器中重新登录...', 'warn');
    setState({ phase: 'login' });
    const loginCtrl = new AbortController();
    await Promise.race([
      page.waitForURL(u => u.toString().includes('member-zone') && !u.toString().includes('log-in'), { timeout: 600000 }),
      stopRace(loginCtrl.signal),
    ]);
    loginCtrl.abort();
    setMsg('✅ 重新登录成功', 'ok');
    const all = await page.context().cookies();
    rt.cookieStr = all.map(c => `${c.name}=${c.value}`).join('; ');
    saveCookies(all as unknown as Parameters<typeof saveCookies>[0]);
    await parseCertsPage(true);
    setState({ phase: 'downloading' });
  } else {
    const all = await page.context().cookies();
    rt.cookieStr = all.map(c => `${c.name}=${c.value}`).join('; ');
    const tok = await page.evaluate(() => (document.getElementById('csrf_token') as HTMLInputElement | null)?.value);
    setState({ csrfToken: tok ?? null });
    const freshCertId = await extractCertId();
    if (freshCertId) {
      setState({ certId: freshCertId });
      const cfg = loadConfig(); cfg.lastCertId = freshCertId; saveConfig(cfg);
    }
  }
}

async function parseCertsPage(skipNav = false): Promise<void> {
  const page = rt.page!;
  if (!skipNav) {
    await page.goto(ASTRILL_CERTS, { waitUntil: 'networkidle', timeout: 30000 });
    await page.waitForTimeout(2000);
  }
  const certId = await extractCertId();
  if (certId) {
    setState({ certId });
    const cfg = loadConfig(); cfg.lastCertId = certId; saveConfig(cfg);
  }
  const csrfToken = await page.evaluate(() => (document.getElementById('csrf_token') as HTMLInputElement | null)?.value);
  const servers = await page.evaluate(() => {
    const sel = document.getElementById('vpn_server') as HTMLSelectElement | null;
    return sel ? Array.from(sel.options).filter(o => o.value !== '0').map(o => ({
      value: o.value, label: o.textContent?.trim() || '', country: o.getAttribute('data-country') || '',
    })) : [];
  });
  setState({ certId: certId ?? null, csrfToken: csrfToken ?? null, servers, phase: 'select' });
  setMsg(`✅ ${servers.length} 个服务器就绪，请选择要下载的服务器`, 'ok');
}

export async function init(): Promise<void> {
  mkdirSync(OVPN_DIR, { recursive: true });
  // Default headless=true for production; set ASTRILL_HEADLESS=false for local debug
  const headless = process.env.ASTRILL_HEADLESS !== 'false';
  rt.browser = await chromium.launch({ headless, slowMo: headless ? 0 : 100 });
  const ctx = await rt.browser.newContext({ acceptDownloads: true });
  rt.page = await ctx.newPage();
  setupCertIdInterception();

  const saved = loadCookies();
  if (saved?.length) {
    await ctx.addCookies(saved as Parameters<typeof ctx.addCookies>[0]);
    rt.cookieStr = saved.map(c => `${c.name}=${c.value}`).join('; ');
  }

  setMsg('🔄 正在打开证书页面...');
  await rt.page.goto(ASTRILL_CERTS, { waitUntil: 'domcontentloaded', timeout: 60000 });
  const url = rt.page.url();

  if (url.includes('log-in')) {
    setMsg('📌 请在浏览器中登录 Astrill...');
    await rt.page.waitForURL(
      u => u.toString().includes('member-zone') && !u.toString().includes('log-in'),
      { timeout: 600000 }
    );
    const all = await ctx.cookies();
    rt.cookieStr = all.map(c => `${c.name}=${c.value}`).join('; ');
    saveCookies(all as unknown as Parameters<typeof saveCookies>[0]);
    await parseCertsPage();
  } else {
    setMsg('✅ 会话有效，跳过登录', 'ok');
    const all = await ctx.cookies();
    rt.cookieStr = all.map(c => `${c.name}=${c.value}`).join('; ');
    saveCookies(all as unknown as Parameters<typeof saveCookies>[0]);
    try { await rt.page.waitForSelector('#vpn_server', { timeout: 10000 }); } catch { /* ignore */ }
    await parseCertsPage(true);
  }
}

export async function startBatch(selectedServers: Array<{ value: string; label: string; country: string }>, mode: string, auto: boolean): Promise<void> {
  const modeVal = mode === 'TCP' ? 'tcp' : 'udp';
  const existingFiles = readdirSync(OVPN_DIR).filter(f => f.endsWith('.ovpn'));

  const jobs = selectedServers.map(srv => {
    const cached = existingFiles.find(f => f.includes(srv.value));
    return { server: srv, captchaB64: null, ocrText: '', status: (cached ? 'cached' : 'pending') as 'cached' | 'pending', file: cached || null, retries: 0 };
  });

  setState({ phase: 'downloading', autoMode: auto, modeVal, downloaded: [], failed: [], jobs, currentIndex: -1 });
  broadcast();

  const page = rt.page!;
  if (!page.url().includes('openvpn-certificates')) {
    await page.goto(ASTRILL_CERTS, { waitUntil: 'networkidle', timeout: 30000 });
  }

  rt.stopRequested = false;
  const state = getState();

  for (let i = 0; i < state.jobs.length; i++) {
    if (rt.stopRequested) {
      state.jobs.slice(i).forEach(j => { if (j.status === 'pending') j.status = 'skipped'; });
      break;
    }
    const job = state.jobs[i];
    if (job.status === 'cached') {
      state.downloaded.push({ label: job.server.label, file: job.file! });
      setMsg(`⏩ [${i + 1}/${state.jobs.length}] ${job.server.label} (已缓存)`);
      state.currentIndex = i;
      broadcast();
      continue;
    }

    let success = false;
    while (job.retries < MAX_RETRY && !success && !rt.stopRequested) {
      state.currentIndex = i;
      job.status = 'captcha';
      setMsg(`📥 [${i + 1}/${state.jobs.length}] ${job.server.label}${job.retries > 0 ? ` (重试 ${job.retries})` : ''}`);
      await ensureLoggedIn();

      try {
        // 1. Fetch captcha via browser context
        const captchaUrl = `https://www.astrill.com/captcha.jpg?time=${Date.now()}&captcha_id=captcha_openvpn_download`;
        job.captchaB64 = await page.evaluate(async (url: string) => {
          const res = await fetch(url, { credentials: 'include' });
          const buf = await res.arrayBuffer();
          const bytes = new Uint8Array(buf);
          let b = '';
          for (let i = 0; i < bytes.length; i++) b += String.fromCharCode(bytes[i]);
          return btoa(b);
        }, captchaUrl);
        broadcast();

        // 2. AI recognize
        job.status = 'recognizing'; broadcast();
        const ocrResult = await recognizeCaptcha(job.captchaB64);
        job.ocrText = ocrResult; broadcast();

        // 3. Get solution
        let solution: string | null;
        if (ocrResult && job.retries < 3) {
          solution = ocrResult;
          setMsg(`🤖 AI 识别: ${ocrResult}，自动提交...`);
        } else {
          job.status = 'waiting'; broadcast();
          const reason = job.retries >= 3 ? `AI 已失败 ${job.retries} 次` : '未识别到验证码';
          setMsg(`✍️ 请输入验证码（${reason}）${ocrResult ? `，AI 建议: ${ocrResult}` : ''}`, 'warn');
          solution = await waitForCaptcha();
        }

        if (!solution) { job.status = 'skipped'; state.failed.push(job.server.label); break; }

        // 4. Download via browser fetch
        job.status = 'downloading'; broadcast();
        const { certId, csrfToken } = getState();
        if (!certId) throw new Error('certId 未获取到');
        if (!csrfToken) throw new Error('csrfToken 未获取到');
        const downloadUrl = `${ASTRILL_CERTS}/download/${encodeURIComponent(certId)}/${encodeURIComponent(job.server.value)}/${encodeURIComponent(modeVal)}/${encodeURIComponent(solution)}/${encodeURIComponent(csrfToken)}`;

        const fileResult = await page.evaluate(async (url: string) => {
          try {
            const res = await fetch(url, { credentials: 'include' });
            const ct = res.headers.get('content-type') || '';
            const cd = res.headers.get('content-disposition') || '';
            const buf = await res.arrayBuffer();
            const bytes = new Uint8Array(buf);
            let b = '';
            for (let i = 0; i < bytes.length; i += 8192) b += String.fromCharCode(...bytes.subarray(i, i + 8192));
            return { ok: true, status: res.status, ct, cd, data: btoa(b), size: bytes.length };
          } catch (e) { return { ok: false, error: (e as Error).message, status: 0, ct: '', cd: '', data: '', size: 0 }; }
        }, downloadUrl);

        if (!fileResult.ok) throw new Error(`fetch 失败: ${fileResult.error}`);
        const fileBytes = Buffer.from(fileResult.data, 'base64');
        const preview = fileBytes.toString('utf-8', 0, 100);
        if (preview.includes('<html') || preview.includes('<!DOCTYPE') || fileResult.size < 100) {
          throw new Error(`服务器返回错误页 (size=${fileResult.size})`);
        }

        let fname = `${job.server.label.replace(/[^a-zA-Z0-9_-]/g, '_')}.ovpn`;
        const cdMatch = fileResult.cd.match(/filename[*]?=["']?([^"';\n]+)/);
        if (cdMatch) {
          // Use basename only to prevent path traversal via Content-Disposition
          const rawName = cdMatch[1].trim();
          const safeName = basename(rawName).replace(/[^a-zA-Z0-9._-]/g, '_');
          if (safeName && safeName.endsWith('.ovpn')) fname = safeName;
        }
        writeFileSync(join(OVPN_DIR, fname), fileBytes);

        job.status = 'done'; job.file = fname; success = true;
        state.downloaded.push({ label: job.server.label, file: fname });
        setMsg(`✅ [${i + 1}/${state.jobs.length}] ${job.server.label} → ${fname}`, 'ok');

      } catch (e) {
        const msg = (e as Error).message;
        if (msg === 'STOPPED' || rt.stopRequested) { job.status = 'skipped'; break; }
        job.retries++;
        if (job.retries >= MAX_RETRY) {
          job.status = 'failed';
          state.failed.push(job.server.label);
          setMsg(`❌ ${job.server.label} 失败: ${msg.substring(0, 80)}`, 'err');
        } else {
          setMsg(`❌ 重试 ${job.server.label} (${job.retries}/${MAX_RETRY})`, 'warn');
          await new Promise(r => setTimeout(r, 500));
        }
      }
      broadcast();
      await new Promise(r => setTimeout(r, 200));
    }
  }

  setState({ phase: 'done', currentIndex: -1 });
  setMsg(`🎉 完成！下载 ${state.downloaded.length} 个，失败 ${state.failed.length} 个`, 'ok');
  broadcast();
}
