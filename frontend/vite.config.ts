import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
  ],
  server: {
    host: '0.0.0.0',
    allowedHosts: true,
    proxy: {
      '/query': 'http://127.0.0.1:8080',
      '/schedule': 'http://127.0.0.1:8080',
      '/plans': 'http://127.0.0.1:8080',
      '/lessons': 'http://127.0.0.1:8080',
    },
  },
})
