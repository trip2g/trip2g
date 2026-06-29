// Deterministic OpenAI-compatible chat server for E2E tests.
// First call to /chat/completions returns a patch_note tool call (with the
// supplied patch arguments); subsequent calls return a finish tool call.
// Serves the path that go-openai hits: <baseURL>/chat/completions.
import http from 'http';

/**
 * Start a deterministic stub LLM server.
 * @param {{ path: string, find: string, replace: string }} patch
 *   Arguments forwarded in the patch_note tool call on the first request.
 * @returns {Promise<{ server: http.Server, port: number, calls: () => number }>}
 */
export function startStubLLM(patch) {
  let calls = 0;
  const server = http.createServer((req, res) => {
    // Only respond to the chat completions endpoint; reject everything else so
    // stray health-check pings do not corrupt the call counter.
    if (!req.url.endsWith('/chat/completions')) {
      res.statusCode = 404;
      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify({ error: { message: 'not found', type: 'invalid_request_error' } }));
      return;
    }

    let body = '';
    req.on('data', (chunk) => (body += chunk));
    req.on('end', () => {
      calls++;
      const toolCall =
        calls === 1
          ? {
              id: 't1',
              type: 'function',
              function: { name: 'patch_note', arguments: JSON.stringify(patch) },
            }
          : {
              id: 't2',
              type: 'function',
              function: { name: 'finish', arguments: JSON.stringify({ answer: 'triaged' }) },
            };

      res.setHeader('Content-Type', 'application/json');
      res.end(
        JSON.stringify({
          id: 'cmpl-stub',
          object: 'chat.completion',
          model: 'stub',
          choices: [
            {
              index: 0,
              message: {
                role: 'assistant',
                content: null,
                tool_calls: [toolCall],
              },
              finish_reason: 'tool_calls',
            },
          ],
          usage: { prompt_tokens: 10, completion_tokens: 5, total_tokens: 15 },
        }),
      );
    });
  });

  return new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      resolve({ server, port: server.address().port, calls: () => calls });
    });
  });
}
