import { getState, addSseClient, removeSseClient, sanitizeForBroadcast, getSseClientCount } from '@/lib/state';

const MAX_SSE_CLIENTS = 10;

export const dynamic = 'force-dynamic';

export async function GET() {
  if (getSseClientCount() >= MAX_SSE_CLIENTS) {
    return new Response('Too many connections', { status: 429 });
  }
  const id = crypto.randomUUID();
  const encoder = new TextEncoder();
  let isClosed = false;

  const stream = new ReadableStream({
    start(controller) {
      function send(data: string) {
        if (isClosed) return;
        try {
          controller.enqueue(encoder.encode(`data:${data}\n\n`));
        } catch {
          isClosed = true;
          removeSseClient(id);
        }
      }
      addSseClient(id, send);
      send(JSON.stringify(sanitizeForBroadcast(getState())));
    },
    cancel() {
      isClosed = true;
      removeSseClient(id);
    },
  });

  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache, no-transform',
      'X-Accel-Buffering': 'no',
      'Connection': 'keep-alive',
    },
  });
}
