import react from '@vitejs/plugin-react'
import { defineConfig, type Plugin } from 'vite'
import { buildRobotsTxt, buildSitemapXml } from './src/lib/siteFiles'

// Link previews (Discord, Bluesky) need absolute URLs and never run the app's JavaScript, so
// they are filled into index.html at build time. Same name as the API's SITE_BASE_URL.
const siteBaseUrl = (process.env.SITE_BASE_URL ?? 'https://fuziongaming.gg').replace(/\/$/, '')

const siteBaseUrlInHtml: Plugin = {
  name: 'site-base-url-in-html',
  transformIndexHtml: (html) => html.replaceAll('{{SITE_BASE_URL}}', siteBaseUrl),
}

const siteFiles: Record<string, { type: string; source: string }> = {
  'robots.txt': { type: 'text/plain', source: buildRobotsTxt(siteBaseUrl) },
  'sitemap.xml': { type: 'application/xml', source: buildSitemapXml(siteBaseUrl) },
}

const crawlerFiles: Plugin = {
  name: 'crawler-files',
  generateBundle() {
    for (const [fileName, { source }] of Object.entries(siteFiles)) {
      this.emitFile({ type: 'asset', fileName, source })
    }
  },
  configureServer(server) {
    server.middlewares.use((request, response, next) => {
      const file = siteFiles[(request.url ?? '').split('?')[0].slice(1)]
      if (!file) return next()
      response.setHeader('Content-Type', file.type)
      response.end(file.source)
    })
  },
}

export default defineConfig({
  plugins: [react(), siteBaseUrlInHtml, crawlerFiles],
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
