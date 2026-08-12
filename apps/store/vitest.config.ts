import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import path from 'path';

// React 19.2.7 in this tree ships act only in the development build
// (react.production.js has no exports.act), and @testing-library/react
// 16.x requires React.act for render().  Force NODE_ENV=development for
// the test process so react resolves to the dev build.  This is a
// pre-existing packaging issue — every render() test crashed with
// "React.act is not a function" before this.
process.env.NODE_ENV = 'development';

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    coverage: {
      reporter: ['text', 'json', 'html'],
      include: ['src/**/*.ts', 'src/**/*.tsx'],
      exclude: ['src/**/*.test.{ts,tsx}', 'src/test/**'],
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
});
