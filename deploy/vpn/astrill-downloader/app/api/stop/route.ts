import { NextResponse } from 'next/server';
import { requestStop } from '@/lib/downloader';
import { setMsg } from '@/lib/state';

export async function POST() {
  requestStop();
  setMsg('⏹️ 正在停止...', 'warn');
  return NextResponse.json({ ok: true });
}
