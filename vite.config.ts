import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import webfontDownload from 'vite-plugin-webfont-dl'

export default defineConfig({
  plugins: [
    vue(),
    webfontDownload('https://fonts.googleapis.com/css2?family=Barlow+Condensed:wght@600;700;800&family=DM+Mono:wght@300;400;500&family=Barlow:wght@300;400;500;600&display=swap'),
  ],
  build: {
    target: 'esnext',
  },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/auth': 'http://127.0.0.1:8080',
      '/admin': 'http://127.0.0.1:8080',
    },
  },
  preview: {
    proxy: {
      '/api': 'http://127.0.0.1:8092',
      '/auth': 'http://127.0.0.1:8092',
      '/admin': 'http://127.0.0.1:8092',
    },
  },
})
