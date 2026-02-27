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
  const body = await req.json() as { apiKey?: string; apiUrl?: string; model?: string };
  const cfg = loadConfig();
  if (body.apiKey !== undefined) cfg.apiKey = body.apiKey;
  if (body.apiUrl !== undefined) cfg.apiUrl = body.apiUrl;
  if (body.model !== undefined) cfg.model = body.model;
  saveConfig(cfg);
  setMsg('✅ AI 配置已保存', 'ok');
  return NextResponse.json({ ok: true });
}
