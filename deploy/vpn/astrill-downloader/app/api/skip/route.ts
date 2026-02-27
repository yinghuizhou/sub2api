import { NextResponse } from 'next/server';
import { resolveCaptcha } from '@/lib/downloader';

export async function POST() {
  resolveCaptcha(null);
  return NextResponse.json({ ok: true });
}
