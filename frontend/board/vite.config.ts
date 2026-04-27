import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  envDir: '../../',
  server: {
    proxy: {
      '/api': {
        target: `http://localhost:${process.env['VITE_BACKEND_PORT'] ?? '8080'}`,
        changeOrigin: true,
        ws: true,
      },
    },
  },
  preview: {
    proxy: {
      '/api': {
        target: `http://localhost:${process.env['VITE_BACKEND_PORT'] ?? '8080'}`,
        changeOrigin: true,
        ws: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.ts', 'src/**/*.test.tsx'],
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
  build: {
    modulePreload: { polyfill: false },
    rollupOptions: {
      output: {
        manualChunks(id) {
          // React core - tiny, cached forever
          if (id.includes('node_modules/react/') || id.includes('node_modules/react-dom/')) {
            return 'vendor-react'
          }
          if (id.includes('node_modules/react-router')) {
            return 'vendor-router'
          }
          if (id.includes('node_modules/@tanstack/')) {
            return 'vendor-query'
          }
          // Zustand + Sonner (tiny state/toast libs)
          if (id.includes('node_modules/zustand') || id.includes('node_modules/sonner')) {
            return 'vendor-ui'
          }
          if (id.includes('node_modules/openapi-fetch') || id.includes('node_modules/openapi-typescript-helpers')) {
            return 'vendor-api'
          }
          // Markdown rendering (react-markdown + rehype-*)
          if (
            id.includes('node_modules/react-markdown') ||
            id.includes('node_modules/rehype') ||
            id.includes('node_modules/remark') ||
            id.includes('node_modules/unified') ||
            id.includes('node_modules/vfile') ||
            id.includes('node_modules/mdast') ||
            id.includes('node_modules/hast') ||
            id.includes('node_modules/unist') ||
            id.includes('node_modules/micromark') ||
            id.includes('node_modules/decode-named-character-reference') ||
            id.includes('node_modules/character-entities')
          ) {
            return 'vendor-markdown'
          }
          // ECharts - already lazy-loaded per page, keep in its own chunk
          if (id.includes('node_modules/echarts') || id.includes('node_modules/zrender')) {
            return 'vendor-echarts'
          }
        },
      },
    },
  },
})
