import { getState, addSseClient, removeSseClient } from '@/lib/state';

export const dynamic = 'force-dynamic';

export async function GET() {
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
      send(JSON.stringify(getState()));
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
