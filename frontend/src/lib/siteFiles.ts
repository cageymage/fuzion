const publicPaths = [
  '/',
  '/news',
  '/roster',
  '/raid-progress',
  '/calendar',
  '/streams',
  '/applications',
  '/professions',
  '/charter',
  '/legal',
]

export function buildRobotsTxt(siteBaseUrl: string): string {
  return [
    'User-agent: *',
    'Allow: /',
    'Disallow: /officer/',
    'Disallow: /dashboard',
    'Disallow: /api/',
    '',
    `Sitemap: ${siteBaseUrl}/sitemap.xml`,
    '',
  ].join('\n')
}

export function buildSitemapXml(siteBaseUrl: string): string {
  return [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
    ...publicPaths.map((path) => `  <url><loc>${siteBaseUrl}${path}</loc></url>`),
    '</urlset>',
    '',
  ].join('\n')
}
