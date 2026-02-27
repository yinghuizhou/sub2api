import { NextRequest, NextResponse } from 'next/server';
import { init, startBatch, getPage } from '@/lib/downloader';
import { setMsg } from '@/lib/state';

export async function POST(req: NextRequest) {
  const { servers, mode, auto } = await req.json() as {
    servers: Array<{ value: string; label: string; country: string }>;
    mode: string;
    auto: boolean;
  };

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
