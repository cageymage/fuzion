import react from '@vitejs/plugin-react'
import { defineConfig, type Plugin } from 'vite'

// Link previews (Discord, Bluesky) need absolute URLs and never run the app's JavaScript, so
// they are filled into index.html at build time. Same name as the API's SITE_BASE_URL.
const siteBaseUrl = (process.env.SITE_BASE_URL ?? 'https://fuziongaming.gg').replace(/\/$/, '')

const siteBaseUrlInHtml: Plugin = {
  name: 'site-base-url-in-html',
  transformIndexHtml: (html) => html.replaceAll('{{SITE_BASE_URL}}', siteBaseUrl),
}

export default defineConfig({
  plugins: [react(), siteBaseUrlInHtml],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
})
