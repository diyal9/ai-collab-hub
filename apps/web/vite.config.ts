import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const hubProxy = env.VITE_HUB_PROXY || 'http://localhost:8085'

  return {
    base: '/ai-collab-hub/',
    build: {
      outDir: 'dist',
    },
    plugins: [vue()],
    server: {
      proxy: {
        '/ai-collab-hub/api': {
          target: hubProxy,
          changeOrigin: true,
        },
        '/api': {
          target: hubProxy,
          changeOrigin: true,
        },
        '/resources': {
          target: hubProxy,
          changeOrigin: true,
        },
        '/ws': {
          target: hubProxy.replace(/^http/, 'ws'),
          ws: true,
        },
      },
    },
  }
})
