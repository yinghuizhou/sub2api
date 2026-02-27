import { NextRequest, NextResponse } from 'next/server';
import { init, startBatch, getPage } from '@/lib/downloader';
import { tryAcquireBatch, releaseBatch, setMsg } from '@/lib/state';

const VALID_SERVER_VALUE = /^[a-zA-Z0-9._-]+$/;

export async function POST(req: NextRequest) {

  let body: { servers: Array<{ value: string; label: string; country: string }>; mode: string; auto: boolean };
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ ok: false, error: 'Invalid JSON body' }, { status: 400 });
  }

  const { servers, mode, auto } = body;
  if (!Array.isArray(servers) || servers.length === 0) {
    return NextResponse.json({ ok: false, error: 'servers must be a non-empty array' }, { status: 400 });
  }
  if (servers.length > 500) {
    return NextResponse.json({ ok: false, error: 'servers must not exceed 500 entries' }, { status: 400 });
  }
  for (const s of servers) {
    if (!s || typeof s.value !== 'string' || typeof s.label !== 'string' || typeof s.country !== 'string') {
      return NextResponse.json({ ok: false, error: 'Each server must have string fields: value, label, country' }, { status: 400 });
    }
    if (s.value.length > 100 || s.label.length > 200 || s.country.length > 100) {
      return NextResponse.json({ ok: false, error: 'Server field length exceeded (value:100, label:200, country:100)' }, { status: 400 });
    }
    if (!VALID_SERVER_VALUE.test(s.value)) {
      return NextResponse.json({ ok: false, error: 'Server value contains invalid characters (allowed: a-z A-Z 0-9 . _ -)' }, { status: 400 });
    }
  }
  if (!['TCP', 'UDP'].includes(String(mode).toUpperCase())) {
    return NextResponse.json({ ok: false, error: 'mode must be TCP or UDP' }, { status: 400 });
  }
  if (typeof auto !== 'boolean') {
    return NextResponse.json({ ok: false, error: 'auto must be a boolean' }, { status: 400 });
  }

  // Atomic check-and-set — prevent concurrent batch runs
  if (!tryAcquireBatch()) {
    return NextResponse.json({ ok: false, error: 'A batch is already running' }, { status: 409 });
  }

  // Fire and forget — run in background
  (async () => {
    try {
      if (!getPage()) {
        setMsg('🚀 正在启动浏览器...');
        await init();
      }
      await startBatch(servers, mode, auto);
    } catch (e) {
      setMsg(`❌ ${(e as Error).message}`, 'err');
      releaseBatch(true);
    }
  })();

  return NextResponse.json({ ok: true });
}
