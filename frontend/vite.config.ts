import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
export default defineConfig({
  plugins: [vue()],
  server: { port: 5173, proxy: { '/api': { target: 'http://localhost:8080', changeOrigin: false } } },
  test: {
    environment: 'jsdom', globals: true, clearMocks: true,
    coverage: {
      provider: 'v8',
      include: ['src/**/*.ts', 'src/**/*.vue'],
      exclude: ['src/generated/api-types.ts'],
      reporter: ['text', 'json', 'json-summary', 'html'],
      reportsDirectory: 'coverage'
    }
  },
  build: { sourcemap: false }
})
