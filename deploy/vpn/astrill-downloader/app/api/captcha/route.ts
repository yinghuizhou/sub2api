import { NextRequest, NextResponse } from 'next/server';
import { resolveCaptcha } from '@/lib/downloader';

export async function POST(req: NextRequest) {
  const { solution } = await req.json() as { solution: string };
  resolveCaptcha(solution);
  return NextResponse.json({ ok: true });
}
