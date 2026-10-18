import { defineConfig } from 'vitest/config';
import { fileURLToPath } from 'node:url';

// Minimum line coverage (%) over the files listed in `coverage.include`.
// Raise it when more tests land; CI fails below this value.
const COVERAGE_THRESHOLD = Number(process.env.FRONTEND_COVERAGE_THRESHOLD ?? 80);

export default defineConfig({
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./', import.meta.url)),
      // The real package throws outside a React Server Components build; tests import server modules directly.
      'server-only': fileURLToPath(new URL('./test/server-only.ts', import.meta.url)),
    },
  },
  test: {
    environment: 'jsdom',
    globals: false,
    globalSetup: ['./test/global-setup.ts'],
    setupFiles: ['./test/setup.ts'],
    include: ['**/*.test.{ts,tsx}'],
    exclude: ['node_modules/**', '.next/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text-summary', 'text', 'lcov'],
      include: ['lib/**/*.{ts,tsx}', 'components/**/*.{ts,tsx}', 'app/**/*.{ts,tsx}'],
      exclude: ['**/*.test.{ts,tsx}', '**/*.d.ts'],
      thresholds: { lines: COVERAGE_THRESHOLD },
    },
  },
});
