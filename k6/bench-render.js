// Page render benchmark: measures per-request latency for public note pages.
//
// Env vars:
//   BASE_URL    — server base URL (default http://localhost:28081)
//   NOTE_COUNT  — number of notes to randomly pick from (default 100)
//   NOTE_PREFIX — URL path prefix for notes (default /bench/note_)
//   VUS         — virtual users (default 10)
//   DURATION    — test duration (default 30s)
import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:28081';
const NOTE_COUNT = parseInt(__ENV.NOTE_COUNT || '100', 10);
const NOTE_PREFIX = __ENV.NOTE_PREFIX || '/bench/note_';
const VUS = parseInt(__ENV.VUS || '10', 10);
const DURATION = __ENV.DURATION || '30s';

export const options = {
  vus: VUS,
  duration: DURATION,
  summaryTrendStats: ['avg', 'p(50)', 'p(95)', 'p(99)', 'min', 'max'],
  thresholds: {
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  const i = Math.floor(Math.random() * NOTE_COUNT);
  const res = http.get(`${BASE_URL}${NOTE_PREFIX}${i}`, {
    tags: { scenario: __ENV.SCENARIO || 'default' },
  });
  check(res, { 'status 200': (r) => r.status === 200 });
}
