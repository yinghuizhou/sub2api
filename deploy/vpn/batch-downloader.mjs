#!/usr/bin/env node
/**
 * Astrill OpenVPN 批量下载器 v4
 * 验证码错误自动重试 + AI 自动识别
 */
import { chromium } from 'playwright';
import http from 'http';
import https from 'https';
import { readFileSync, mkdirSync, writeFileSync, existsSync, readdirSync } from 'fs';
import { join, resolve } from 'path';
import { execSync } from 'child_process';

const DIR = resolve(import.meta.dirname, 'ovpn-configs');
const UI_PATH = resolve(import.meta.dirname, 'batch-downloader-ui.html');
const COOKIES_FILE = resolve(import.meta.dirname, '.astrill-cookies.json');
const CONFIG_FILE = resolve(import.meta.dirname, '.ai-config.json');
const PORT = 3457;
const ASTRILL_LOGIN = 'https://www.astrill.com/member-zone/log-in';
const ASTRILL_CERTS = 'https://www.astrill.com/member-zone/tools/openvpn-certificates';
const CAPTCHA_URL = 'https://www.astrill.com/captcha.jpg';
const MAX_RETRY = 3;

let state = { phase: 'login', servers: [], certId: null, csrfToken: null, current: null,
  captchaB64: null, ocrText: '', downloaded: [], failed: [], message: '', autoMode: false, retryCount: 0 };
let cookies = '';
let page, browser;
const sseClients = [];
let captchaResolve = null;
let stopRequested = false;

// 网络请求拦截：自动从请求/响应 URL 中捕获 certId
function setupCertIdInterception() {
  page.on('request', req => {
    const m = req.url().match(/\/openvpn-certificates\/download\/(\d{5,12})\//);
    if (m && !state.certId) {
      state.certId = m[1];
      console.log(`🔑 certId captured from request URL: ${state.certId}`);
    }
  });
  page.on('response', async res => {
    if (state.certId) return;
    const url = res.url();
    const m = url.match(/\/openvpn-certificates\/download\/(\d{5,12})\//);
    if (m) { state.certId = m[1]; console.log(`🔑 certId captured from response URL: ${state.certId}`); return; }
    // 检查 JSON 响应中是否包含 certId
    if (url.includes('astrill.com') && res.headers()['content-type']?.includes('json')) {
      try {
        const text = await res.text().catch(() => '');
        const jm = text.match(/["']?cert[_-]?id["']?\s*[:=]\s*["']?(\d{5,12})["']?/i);
        if (jm) { state.certId = jm[1]; console.log(`🔑 certId found in JSON response: ${state.certId}`); }
      } catch {}
    }
  });
}

function broadcast() { const d = `data:${JSON.stringify(state)}\n\n`; sseClients.forEach(r => r.write(d)); }
function setMsg(m) { state.message = m; broadcast(); console.log(m); }
function readBody(req) { return new Promise(r => { let b = ''; req.on('data', c => b += c); req.on('end', () => r(b)); }); }
function waitForCaptcha() { return new Promise(r => { captchaResolve = r; }); }

async function closeModal() {
  try {
    await page.getByText('关闭窗口').click().catch(() => {});
    await page.waitForSelector('.modal.show, .modal[style*="display: block"], .modal.in', {
      state: 'hidden', timeout: 3000
    }).catch(() => {});
    await page.waitForTimeout(300);
  } catch {}
}

async function extractCertId() {
  // 如果之前网络拦截已经捕获了，直接用
  if (state.certId) return state.certId;

  let certId = await page.evaluate(() => {
    // 1. 所有带 onclick 的元素（不限于 <a>）
    for (const el of document.querySelectorAll('[onclick]')) {
      const m = (el.getAttribute('onclick') || '').match(/(\d{5,12})/);
      if (m && el.getAttribute('onclick').includes('Download')) return m[1];
    }
    // 2. 所有内联 script 标签文本
    for (const s of document.querySelectorAll('script:not([src])')) {
      const m = s.textContent.match(/OnDownloadSingle\(['"](\d+)['"]\)/);
      if (m) return m[1];
      // certId = 28724154 / cert_id: 28724154
      const m2 = s.textContent.match(/cert[_-]?id['":\s]*[=:]\s*['"]?(\d{5,12})['"]?/i);
      if (m2) return m2[1];
    }
    // 3. data 属性
    for (const el of document.querySelectorAll('[data-cert-id],[data-certid],[data-certificate-id]')) {
      const v = el.getAttribute('data-cert-id') || el.getAttribute('data-certid') || el.getAttribute('data-certificate-id');
      if (v && /^\d{5,12}$/.test(v)) return v;
    }
    // 4. window 全局变量
    for (const key of ['certId','cert_id','CERT_ID','certificateId','certificate_id']) {
      if (window[key] && /^\d{5,12}$/.test(String(window[key]))) return String(window[key]);
    }
    // 5. hidden input
    const inp = document.querySelector('input[name="cert_id"],input[name="certId"]');
    if (inp?.value && /^\d{5,12}$/.test(inp.value)) return inp.value;
    // 6. 全页面 innerHTML 最后扫描
    const m = document.documentElement.innerHTML.match(/\/download\/(\d{5,12})\//);
    if (m) return m[1];
    const m2 = document.documentElement.innerHTML.match(/OnDownloadSingle\(['"](\d+)['"]\)/);
    if (m2) return m2[1];
    return null;
  });

  // 如果页面 JS 扫描无果，尝试打开 Modal 提取
  if (!certId) {
    try { certId = await extractCertIdFromModal(); } catch (e) {
      console.log(`⚠️ Modal extraction failed: ${e.message}`);
    }
  }

  // 最后兜底：使用上次成功缓存的 certId
  if (!certId) {
    try {
      const cfg = loadConfig();
      if (cfg.lastCertId) {
        console.log(`🔑 certId fallback to cached: ${cfg.lastCertId}`);
        certId = cfg.lastCertId;
      }
    } catch {}
  }

  if (certId) console.log(`🔑 certId found: ${certId}`);
  else console.log(`⚠️ certId not found on page`);
  return certId;
}

async function extractCertIdFromModal() {
  // 找到"下载"相关按钮，点击打开 Modal
  const candidates = [
    page.locator('text=下载个人资料').first(),
    page.locator('button:has-text("下载")').first(),
    page.locator('[onclick*="Download"]').first(),
    page.locator('[onclick*="download"]').first(),
  ];
  let clicked = false;
  for (const btn of candidates) {
    if (await btn.isVisible({ timeout: 1000 }).catch(() => false)) {
      // 在点击前设置一次性 request listener
      let captured = null;
      const handler = req => {
        const m = req.url().match(/\/download\/(\d{5,12})\//);
        if (m) captured = m[1];
      };
      page.on('request', handler);
      await btn.click();
      await page.waitForTimeout(1500);
      page.off('request', handler);
      if (captured) { await closeModal(); return captured; }
      // 扫描 Modal HTML
      const certId = await page.evaluate(() => {
        const modal = document.querySelector('.modal.show, .modal[style*="block"], .modal.in');
        const src = (modal || document.documentElement).innerHTML;
        const m = src.match(/\/download\/(\d{5,12})\//);
        if (m) return m[1];
        const m2 = src.match(/OnDownloadSingle\(['"](\d+)['"]\)/);
        if (m2) return m2[1];
        const m3 = src.match(/cert[_-]?id['":\s]*[=:]\s*['"]?(\d{5,12})['"]?/i);
        if (m3) return m3[1];
        return null;
      });
      await closeModal();
      if (certId) return certId;
      clicked = true;
      break;
    }
  }
  if (!clicked) throw new Error('未找到可点击的下载按钮');
  throw new Error('Modal 中未找到 certId');
}

function stopRace() {
  return new Promise((_, reject) => {
    const t = setInterval(() => { if (stopRequested) { clearInterval(t); reject(new Error('STOPPED')); } }, 200);
  });
}

async function ensureLoggedIn() {
  if (stopRequested) throw new Error('STOPPED');
  await Promise.race([
    page.goto(ASTRILL_CERTS, { waitUntil: 'domcontentloaded', timeout: 30000 }),
    stopRace()
  ]);
  await Promise.race([
    page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => {}),
    stopRace()
  ]);
  const url = page.url();
  console.log(`🔑 Session check: ${url}`);
  if (url.includes('log-in')) {
    setMsg('🔑 会话已过期，请在浏览器中重新登录...');
    state.phase = 'login'; broadcast();
    // 等待登录，同时支持停止
    await Promise.race([
      page.waitForURL(u => u.toString().includes('member-zone') && !u.toString().includes('log-in'), { timeout: 600000 }),
      new Promise((_, reject) => {
        const t = setInterval(() => { if (stopRequested) { clearInterval(t); reject(new Error('STOPPED')); } }, 300);
      })
    ]);
    setMsg('✅ 重新登录成功，正在刷新 session...');
    const allCookies = await page.context().cookies();
    cookies = allCookies.map(c => `${c.name}=${c.value}`).join('; ');
    saveCookies(allCookies);
    try { await page.waitForSelector('#vpn_server', { timeout: 10000 }); } catch {}
    await parseCertsPage(true);
    state.phase = 'downloading'; broadcast();
  } else {
    const allCookies = await page.context().cookies();
    cookies = allCookies.map(c => `${c.name}=${c.value}`).join('; ');
    state.csrfToken = await page.evaluate(() => document.getElementById('csrf_token')?.value);
    const freshCertId = await extractCertId();
    if (freshCertId) {
      state.certId = freshCertId;
      const cfg = loadConfig(); cfg.lastCertId = freshCertId; saveConfig(cfg);
    }
    console.log(`🔑 Session refreshed — certId: ${state.certId}, csrfToken: ${state.csrfToken?.substring(0, 8)}...`);
  }
}

function httpsReq(url, opts = {}) {
  return new Promise((resolve, reject) => {
    const u = new URL(url);
    const bodyBuf = opts.body ? Buffer.from(opts.body) : null;
    const headers = { 'User-Agent': 'Mozilla/5.0', ...opts.headers };
    if (bodyBuf) headers['Content-Length'] = bodyBuf.length;
    const req = https.request({
      hostname: u.hostname, path: u.pathname + u.search, method: opts.method || 'GET',
      headers, timeout: opts.timeout || 30000
    }, res => {
      const chunks = []; res.on('data', c => chunks.push(c));
      res.on('end', () => resolve({ status: res.statusCode, headers: res.headers, body: Buffer.concat(chunks) }));
    });
    req.on('timeout', () => { req.destroy(); reject(new Error(`Request timeout: ${url}`)); });
    req.on('error', reject);
    if (bodyBuf) req.write(bodyBuf);
    req.end();
  });
}

function fetchAstrill(url) {
  return httpsReq(url, { headers: { Cookie: cookies } });
}

function loadConfig() {
  try { if (existsSync(CONFIG_FILE)) return JSON.parse(readFileSync(CONFIG_FILE, 'utf-8')); } catch {}
  return {};
}
function saveConfig(cfg) {
  try { writeFileSync(CONFIG_FILE, JSON.stringify(cfg, null, 2)); } catch {}
}
function getApiKey() { return loadConfig().apiKey || ''; }

// AI CAPTCHA Recognition via sub2api proxy
function httpReq(url, opts = {}) {
  const mod = url.startsWith('https') ? https : http;
  return new Promise((resolve, reject) => {
    const u = new URL(url);
    const bodyBuf = opts.body ? Buffer.from(opts.body) : null;
    const headers = { 'User-Agent': 'Mozilla/5.0', ...opts.headers };
    if (bodyBuf) headers['Content-Length'] = bodyBuf.length;
    const req = mod.request({
      hostname: u.hostname, port: u.port || (u.protocol === 'https:' ? 443 : 80),
      path: u.pathname + u.search, method: opts.method || 'GET',
      headers, timeout: opts.timeout || 30000
    }, res => {
      const chunks = []; res.on('data', c => chunks.push(c));
      res.on('end', () => resolve({ status: res.statusCode, headers: res.headers, body: Buffer.concat(chunks) }));
    });
    req.on('timeout', () => { req.destroy(); reject(new Error(`Request timeout: ${url}`)); });
    req.on('error', reject);
    if (bodyBuf) req.write(bodyBuf);
    req.end();
  });
}

async function recognizeCaptcha(imgBase64) {
  const cfg = loadConfig();
  const apiUrl = cfg.apiUrl || '';
  const apiKey = cfg.apiKey || '';
  if (!apiUrl || !apiKey) { setMsg('⚠️ 未配置 AI 接口，跳过自动识别'); return ''; }
  try {
    const endpoint = apiUrl.replace(/\/+$/, '') + '/v1/messages';
    const body = JSON.stringify({
      model: cfg.model || 'claude-haiku-4-5-20251001', max_tokens: 50,
      messages: [{ role: 'user', content: [
        { type: 'image', source: { type: 'base64', media_type: 'image/jpeg', data: imgBase64 } },
        { type: 'text', text: 'What letters/characters appear in this image? Output only those exact characters, preserving case. No explanation, just the characters.' }
      ]}]
    });
    console.log(`🤖 Calling ${endpoint} (body size: ${body.length} bytes)...`);
    const res = await httpReq(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'x-api-key': apiKey, 'anthropic-version': '2023-06-01' },
      body, timeout: 15000
    });
    const raw = res.body.toString();
    console.log(`🤖 API response (status: ${res.status}): ${raw.substring(0, 300)}`);
    const data = JSON.parse(raw);
    if (data.error) { console.log('🤖 API error:', data.error.message); setMsg(`⚠️ AI 识别失败: ${data.error.message}`); return ''; }
    if (data.content?.[0]?.text) return data.content[0].text.trim();
  } catch (e) { console.log('🤖 OCR error:', e.message); setMsg(`⚠️ AI 识别出错: ${e.message}`); }
  return '';
}

// HTTP Server
const server = http.createServer(async (req, res) => {
  const h = { 'Access-Control-Allow-Origin': '*', 'Access-Control-Allow-Methods': '*', 'Access-Control-Allow-Headers': '*' };
  if (req.method === 'OPTIONS') { res.writeHead(200, h); return res.end(); }
  if (req.url === '/') { res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' }); res.end(readFileSync(UI_PATH)); }
  else if (req.url === '/events') {
    res.writeHead(200, { ...h, 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache', Connection: 'keep-alive' });
    sseClients.push(res); res.write(`data:${JSON.stringify(state)}\n\n`);
    req.on('close', () => { const i = sseClients.indexOf(res); if (i >= 0) sseClients.splice(i, 1); });
  } else if (req.url === '/api/start' && req.method === 'POST') {
    const { servers, mode, auto } = JSON.parse(await readBody(req));
    res.writeHead(200, h); res.end('ok');
    startBatch(servers, mode, auto);
  } else if (req.url === '/api/captcha' && req.method === 'POST') {
    const { solution } = JSON.parse(await readBody(req));
    res.writeHead(200, h); res.end('ok');
    if (captchaResolve) captchaResolve(solution);
  } else if (req.url === '/api/skip' && req.method === 'POST') {
    res.writeHead(200, h); res.end('ok');
    if (captchaResolve) captchaResolve(null);
  } else if (req.url === '/api/stop' && req.method === 'POST') {
    res.writeHead(200, h); res.end('ok');
    if (stopRequested) return; // 防止重复触发
    stopRequested = true;
    if (captchaResolve) captchaResolve(null);
    setMsg('⏹️ 正在停止...');
    // 中止当前页面导航（等价于按 Escape 键）
    page?.evaluate(() => window.stop()).catch(() => {});
  } else if (req.url === '/api/config' && req.method === 'GET') {
    const cfg = loadConfig();
    const key = cfg.apiKey || '';
    const masked = key.length > 14 ? key.slice(0, 10) + '...' + key.slice(-4) : key;
    res.writeHead(200, { ...h, 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ apiKey: masked, apiUrl: cfg.apiUrl || '', model: cfg.model || '', hasKey: !!key }));
  } else if (req.url === '/api/config' && req.method === 'POST') {
    const body = JSON.parse(await readBody(req));
    const cfg = loadConfig();
    if (body.apiKey !== undefined) cfg.apiKey = body.apiKey;
    if (body.apiUrl !== undefined) cfg.apiUrl = body.apiUrl;
    if (body.model !== undefined) cfg.model = body.model;
    saveConfig(cfg);
    res.writeHead(200, h); res.end('ok');
    setMsg(`✅ AI 配置已保存`);
  } else { res.writeHead(404, h); res.end(); }
});

function saveCookies(allCookies) {
  try { writeFileSync(COOKIES_FILE, JSON.stringify(allCookies, null, 2)); } catch {}
}

function loadCookies() {
  try {
    if (existsSync(COOKIES_FILE)) return JSON.parse(readFileSync(COOKIES_FILE, 'utf-8'));
  } catch {}
  return null;
}

async function restoreCookies(ctx) {
  const saved = loadCookies();
  if (!saved?.length) { console.log('🔑 No saved cookies found'); return; }
  console.log(`🔑 Restoring ${saved.length} cookies`);
  await ctx.addCookies(saved);
  cookies = saved.map(c => `${c.name}=${c.value}`).join('; ');
}

async function parseCertsPage(skipNav = false) {
  if (!skipNav) {
    await page.goto(ASTRILL_CERTS, { waitUntil: 'networkidle', timeout: 30000 });
    await page.waitForTimeout(2000);
  }
  state.certId = await extractCertId();
  if (state.certId) {
    const cfg = loadConfig(); cfg.lastCertId = state.certId; saveConfig(cfg);
  }
  state.csrfToken = await page.evaluate(() => document.getElementById('csrf_token')?.value);
  state.servers = await page.evaluate(() => {
    const sel = document.getElementById('vpn_server');
    return sel ? Array.from(sel.options).filter(o => o.value !== '0').map(o => ({
      value: o.value, label: o.textContent.trim(), country: o.getAttribute('data-country') || ''
    })) : [];
  });
  state.phase = 'select';
  setMsg(`✅ ${state.servers.length} 个服务器就绪，请选择要下载的服务器`);
}

async function init() {
  mkdirSync(DIR, { recursive: true });
  browser = await chromium.launch({ headless: false, slowMo: 100 });
  const ctx = await browser.newContext({ acceptDownloads: true });
  page = await ctx.newPage();
  setupCertIdInterception(); // 立即设置网络拦截，捕获 certId

  // Restore saved cookies and go directly to certs page
  await restoreCookies(ctx);
  setMsg('🔄 正在打开证书页面...');
  await page.goto(ASTRILL_CERTS, { waitUntil: 'domcontentloaded', timeout: 60000 });

  // Check where we landed
  const url = page.url();
  console.log(`🔑 Landed on: ${url}`);

  if (url.includes('log-in')) {
    // Redirected to login — wait for user to log in
    setMsg('📌 请在浏览器中登录 Astrill...');
    await page.waitForURL(u => u.toString().includes('member-zone') && !u.toString().includes('log-in'), { timeout: 600000 });
    setMsg('✅ 登录成功！正在获取数据...');
    const allCookies = await ctx.cookies();
    cookies = allCookies.map(c => `${c.name}=${c.value}`).join('; ');
    saveCookies(allCookies);
    await parseCertsPage();
  } else {
    // Already on certs page — session is valid
    setMsg('✅ 会话有效，跳过登录');
    const allCookies = await ctx.cookies();
    cookies = allCookies.map(c => `${c.name}=${c.value}`).join('; ');
    saveCookies(allCookies);
    // Wait for full load then parse
    try { await page.waitForSelector('#vpn_server', { timeout: 10000 }); } catch {}
    await parseCertsPage(true);
  }
}

async function startBatch(selectedServers, mode, auto) {
  state.phase = 'downloading';
  state.autoMode = auto || false;
  state.modeVal = mode === 'TCP' ? 'tcp' : 'udp';
  state.downloaded = [];
  state.failed = [];
  // jobs: status = pending|captcha|recognizing|downloading|done|failed|skipped|waiting
  // File cache: check existing .ovpn files
  const existingFiles = readdirSync(DIR).filter(f => f.endsWith('.ovpn'));

  state.jobs = selectedServers.map(srv => {
    const cached = existingFiles.find(f => f.includes(srv.value));
    return {
      server: srv, captchaB64: null, ocrText: '',
      status: cached ? 'cached' : 'pending', file: cached || null, retries: 0
    };
  });
  broadcast();

  // Ensure we're on the certs page
  if (!page.url().includes('openvpn-certificates')) {
    await page.goto(ASTRILL_CERTS, { waitUntil: 'networkidle', timeout: 30000 });
  }

  stopRequested = false;
  for (let i = 0; i < state.jobs.length; i++) {
    if (stopRequested) {
      state.jobs.slice(i).forEach(j => { if (j.status === 'pending') j.status = 'skipped'; });
      break;
    }
    const job = state.jobs[i];
    if (job.status === 'cached') {
      state.downloaded.push({ label: job.server.label, file: job.file });
      setMsg(`⏩ [${i + 1}/${state.jobs.length}] ${job.server.label} (已缓存: ${job.file})`);
      state.currentIndex = i;
      broadcast();
      continue;
    }

    let success = false;
    while (job.retries < MAX_RETRY && !success && !stopRequested) {
      state.currentIndex = i;
      job.status = 'captcha';
      setMsg(`📥 [${i + 1}/${state.jobs.length}] ${job.server.label}${job.retries > 0 ? ` (重试 ${job.retries})` : ''}`);

      // Verify session is still valid before each download attempt
      await ensureLoggedIn();

      try {
        // 1. Fetch captcha via browser's fetch (keeps session/cookies identical)
        const captchaUrl = `https://www.astrill.com/captcha.jpg?time=${Date.now()}&captcha_id=captcha_openvpn_download`;
        job.captchaB64 = await page.evaluate(async (url) => {
          const res = await fetch(url, { credentials: 'include' });
          const buf = await res.arrayBuffer();
          const bytes = new Uint8Array(buf);
          let b = '';
          for (let i = 0; i < bytes.length; i++) b += String.fromCharCode(bytes[i]);
          return btoa(b);
        }, captchaUrl);
        broadcast();

        // 2. AI recognize
        job.status = 'recognizing';
        broadcast();
        const ocrResult = await recognizeCaptcha(job.captchaB64);
        job.ocrText = ocrResult;
        broadcast();

        // 3. Get solution: 前3次自动用AI，第4次起转为手动
        let solution;
        if (ocrResult && job.retries < 3) {
          solution = ocrResult;
          setMsg(`🤖 AI 识别: ${ocrResult}，自动提交...`);
        } else {
          job.status = 'waiting';
          broadcast();
          const reason = job.retries >= 3 ? `AI 已失败 ${job.retries} 次，请手动输入` : '未识别到验证码';
          setMsg(`请输入验证码（${reason}）${ocrResult ? `，AI 建议: ${ocrResult}` : ''}`);
          solution = await waitForCaptcha();
        }

        if (!solution) {
          job.status = 'skipped';
          state.failed.push(job.server.label);
          break;
        }

        // 4. Fetch download URL directly in browser context (same session, same origin, auto Referer)
        job.status = 'downloading';
        broadcast();
        if (!state.certId) throw new Error('certId 未获取到，请检查 Astrill 页面是否已加载证书信息');
        const downloadUrl = `${ASTRILL_CERTS}/download/${state.certId}/${job.server.value}/${state.modeVal}/${encodeURIComponent(solution)}/${state.csrfToken}`;
        console.log(`📡 Download URL: ${downloadUrl}`);

        const fileResult = await page.evaluate(async (url) => {
          try {
            const res = await fetch(url, { credentials: 'include' });
            const ct = res.headers.get('content-type') || '';
            const cd = res.headers.get('content-disposition') || '';
            const buf = await res.arrayBuffer();
            const bytes = new Uint8Array(buf);
            let b = '';
            // Convert to base64 in chunks to avoid stack overflow
            for (let i = 0; i < bytes.length; i += 8192) {
              b += String.fromCharCode(...bytes.subarray(i, i + 8192));
            }
            return { ok: true, status: res.status, ct, cd, data: btoa(b), size: bytes.length };
          } catch (e) {
            return { ok: false, error: e.message };
          }
        }, downloadUrl);

        console.log(`📡 Response: status=${fileResult.status} ct=${fileResult.ct} cd=${fileResult.cd} size=${fileResult.size}`);

        if (!fileResult.ok) throw new Error(`fetch 失败: ${fileResult.error}`);

        // Check if server returned an error page (HTML) instead of file
        const fileBytes = Buffer.from(fileResult.data, 'base64');
        const preview = fileBytes.toString('utf-8', 0, 100);
        if (preview.includes('<html') || preview.includes('<!DOCTYPE') || fileResult.size < 100) {
          throw new Error(`服务器返回错误页 (size=${fileResult.size}): ${preview.substring(0, 150)}`);
        }

        // Extract filename from Content-Disposition or use label
        let fname = `${job.server.label.replace(/[^a-zA-Z0-9_-]/g, '_')}.ovpn`;
        const cdMatch = fileResult.cd.match(/filename[*]?=["']?([^"';\n]+)/);
        if (cdMatch) fname = cdMatch[1].trim();
        writeFileSync(join(DIR, fname), fileBytes);

        job.status = 'done';
        job.file = fname;
        success = true;
        state.downloaded.push({ label: job.server.label, file: fname });
        setMsg(`✅ [${i + 1}/${state.jobs.length}] ${job.server.label} → ${fname}`);

      } catch (e) {
        if (e.message === 'STOPPED' || stopRequested) {
          job.status = 'skipped'; break;
        }
        console.log(`❌ Download error [${job.server.label}]: ${e.message}`);
        job.retries++;
        if (job.retries >= MAX_RETRY) {
          job.status = 'failed';
          state.failed.push(job.server.label);
          setMsg(`❌ ${job.server.label} 失败: ${e.message.substring(0, 80)}`);
        } else {
          setMsg(`❌ 重试 ${job.server.label} (${job.retries}/${MAX_RETRY})`);
          await new Promise(r => setTimeout(r, 500));
        }
      }
      broadcast();
      await new Promise(r => setTimeout(r, 200));
    }
  }

  state.phase = 'done';
  state.currentIndex = -1;
  setMsg(`🎉 完成！下载 ${state.downloaded.length} 个，失败 ${state.failed.length} 个`);
  broadcast();
}

server.listen(PORT, () => {
  console.log(`\n🌐 控制面板: http://localhost:${PORT}\n`);
  try { execSync(`open "http://localhost:${PORT}"`); } catch {}
  init().catch(e => { setMsg(`❌ ${e.message}`); });
});
