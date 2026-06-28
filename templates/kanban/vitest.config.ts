import { defineConfig } from 'vitest/config'

// Integration tests render <Board> in jsdom with a mocked ./api module. The pure
// node:test suite (src/smoke.test.ts) keeps its own runner; vitest only owns the
// *.integration.test.tsx files so the two test styles don't collide.
export default defineConfig({
  test: {
    include: ['src/**/*.integration.test.{ts,tsx}'],
    environment: 'jsdom',
    globals: true,
    setupFiles: ['src/test-setup.ts'],
  },
})
