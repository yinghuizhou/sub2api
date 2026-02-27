import { NextRequest, NextResponse } from 'next/server';
import { loadConfig, saveConfig } from '@/lib/config';
import { setMsg } from '@/lib/state';

export async function GET() {
  const cfg = loadConfig();
  const key = cfg.apiKey || '';
  const masked = key.length > 14 ? key.slice(0, 10) + '...' + key.slice(-4) : key;
  return NextResponse.json({
    apiKey: masked,
    apiUrl: cfg.apiUrl || '',
    model: cfg.model || '',
    hasKey: !!key,
  });
}

export async function POST(req: NextRequest) {
  let body: { apiKey?: unknown; apiUrl?: unknown; model?: unknown };
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ ok: false, error: 'Invalid JSON body' }, { status: 400 });
  }
  const cfg = loadConfig();
  if (body.apiKey !== undefined) {
    const key = String(body.apiKey);
    if (key.length > 500) return NextResponse.json({ ok: false, error: 'apiKey too long' }, { status: 400 });
    cfg.apiKey = key;
  }
  if (body.apiUrl !== undefined) {
    const url = String(body.apiUrl);
    if (url.length > 300) return NextResponse.json({ ok: false, error: 'apiUrl too long' }, { status: 400 });
    if (url && !url.startsWith('https://')) return NextResponse.json({ ok: false, error: 'apiUrl must start with https://' }, { status: 400 });
    cfg.apiUrl = url;
  }
  if (body.model !== undefined) {
    const model = String(body.model);
    if (model.length > 200) return NextResponse.json({ ok: false, error: 'model too long' }, { status: 400 });
    cfg.model = model;
  }
  saveConfig(cfg);
  setMsg('✅ AI 配置已保存', 'ok');
  return NextResponse.json({ ok: true });
}
