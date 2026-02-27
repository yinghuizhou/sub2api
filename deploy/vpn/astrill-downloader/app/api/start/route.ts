import { NextRequest, NextResponse } from 'next/server';
import { init, startBatch, getPage } from '@/lib/downloader';
import { getState, setMsg } from '@/lib/state';

export async function POST(req: NextRequest) {
  // Prevent concurrent batch runs
  const { phase } = getState();
  if (phase === 'downloading') {
    return NextResponse.json({ ok: false, error: 'A batch is already running' }, { status: 409 });
  }

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
  if (!['TCP', 'UDP'].includes(String(mode).toUpperCase())) {
    return NextResponse.json({ ok: false, error: 'mode must be TCP or UDP' }, { status: 400 });
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
    }
  })();

  return NextResponse.json({ ok: true });
}
