import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: '/hermesshare/',
  build: {
    outDir: 'dist'
  },
  plugins: [vue()],
  server: { proxy: { '/api': 'http://localhost:8085', '/ws': { target: 'ws://localhost:8085', ws: true } } }
})
