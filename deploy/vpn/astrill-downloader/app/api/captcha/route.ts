import { NextRequest, NextResponse } from 'next/server';
import { resolveCaptcha } from '@/lib/downloader';

export async function POST(req: NextRequest) {
  let body: { solution?: unknown };
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ ok: false, error: 'Invalid JSON body' }, { status: 400 });
  }

  const { solution } = body;
  if (typeof solution !== 'string' || solution.length === 0 || solution.length > 20) {
    return NextResponse.json({ ok: false, error: 'solution must be a non-empty string up to 20 chars' }, { status: 400 });
  }

  resolveCaptcha(solution);
  return NextResponse.json({ ok: true });
}
