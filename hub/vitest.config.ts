import { defineConfig, mergeConfig } from 'vitest/config'
import viteConfig from './vite.config'

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: 'jsdom',
      setupFiles: ['./src/setupTests.ts'],
      css: true,
      env: { VITE_API_BASE_URL: 'http://localhost:3000/api' },
      restoreMocks: true,
    },
  }),
)
