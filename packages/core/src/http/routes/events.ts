import type { Route } from "../router.ts";

const encoder = new TextEncoder();

export const eventRoutes: readonly Route[] = [
  {
    method: "GET",
    path: "/events",
    queryToken: true,
    handler: ({ hub }) => {
      let unsubscribe = (): void => {};
      const stream = new ReadableStream<Uint8Array>({
        start(controller) {
          const send = (chunk: string): void => controller.enqueue(encoder.encode(chunk));
          send(": connected\n\n");
          unsubscribe = hub.subscribe(send);
        },
        cancel: () => unsubscribe(),
      });
      return new Response(stream, {
        headers: { "content-type": "text/event-stream", "cache-control": "no-cache", connection: "keep-alive" },
      });
    },
  },
];
