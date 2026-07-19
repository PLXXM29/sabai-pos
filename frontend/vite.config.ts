import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

// Backend API base (dev). Proxying /api keeps the app same-origin with the API
// so the httpOnly refresh cookie works and there's no CORS.
const API_TARGET = process.env.VITE_API_TARGET ?? 'http://localhost:8082'

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        globPatterns: ['**/*.{js,css,html,woff2}'],
        navigateFallbackDenylist: [/^\/api/],
      },
      manifest: {
        name: 'MiniMart POS',
        short_name: 'MiniMart',
        description: 'ระบบขายหน้าร้านมินิมาร์ท (offline-first)',
        lang: 'th',
        theme_color: '#0F3B39',
        background_color: '#F7F1E6',
        display: 'standalone',
        start_url: '/',
        icons: [
          { src: '/icon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any maskable' },
        ],
      },
    }),
  ],
  server: {
    port: 5173,
    host: true,
    proxy: {
      '/api': { target: API_TARGET, changeOrigin: true },
    },
  },
})
