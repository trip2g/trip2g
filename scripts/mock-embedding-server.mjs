// Minimal OpenAI-compatible embedding server returning random unit vectors.
// Used by the pushNotes benchmark to measure vector index memory without
// waiting for a real model (which saturates at ~1000+ notes).
//
// Usage: node scripts/mock-embedding-server.mjs [port]   # default 11435
import http from 'node:http';

const PORT = parseInt(process.argv[2] || '11435', 10);
const DIMS = 1024;

function randVec() {
  const v = Array.from({ length: DIMS }, () => Math.random() * 2 - 1);
  const norm = Math.sqrt(v.reduce((s, x) => s + x * x, 0));
  return v.map((x) => x / norm);
}

const server = http.createServer((req, res) => {
  if (req.method !== 'POST' || req.url !== '/v1/embeddings') {
    res.writeHead(404);
    res.end('not found');
    return;
  }
  let body = '';
  req.on('data', (chunk) => { body += chunk; });
  req.on('end', () => {
    try {
      const { input, model } = JSON.parse(body);
      const inputs = Array.isArray(input) ? input : [input];
      const data = inputs.map((_, i) => ({ object: 'embedding', index: i, embedding: randVec() }));
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ object: 'list', data, model: model || 'mock', usage: { prompt_tokens: 0, total_tokens: 0 } }));
    } catch (e) {
      res.writeHead(400);
      res.end(String(e));
    }
  });
});

server.listen(PORT, '127.0.0.1', () => {
  process.stderr.write(`mock-embedding-server listening on 127.0.0.1:${PORT}\n`);
});
