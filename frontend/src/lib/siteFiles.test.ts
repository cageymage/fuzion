import { describe, expect, it } from 'vitest'
import { buildRobotsTxt, buildSitemapXml } from './siteFiles'

describe('buildRobotsTxt', () => {
  it('should point at the sitemap with an absolute url when given a base url', () => {
    expect(buildRobotsTxt('https://fuziongaming.gg')).toContain(
      'Sitemap: https://fuziongaming.gg/sitemap.xml',
    )
  })

  it('should allow crawlers everywhere except officer, dashboard and api paths', () => {
    expect(buildRobotsTxt('https://fuziongaming.gg')).toBe(
      [
        'User-agent: *',
        'Allow: /',
        'Disallow: /officer/',
        'Disallow: /dashboard',
        'Disallow: /api/',
        '',
        'Sitemap: https://fuziongaming.gg/sitemap.xml',
        '',
      ].join('\n'),
    )
  })
})

describe('buildSitemapXml', () => {
  it('should list the public pages as absolute urls when given a base url', () => {
    const xml = buildSitemapXml('https://fuziongaming.gg')

    expect(xml).toBe(
      [
        '<?xml version="1.0" encoding="UTF-8"?>',
        '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
        '  <url><loc>https://fuziongaming.gg/</loc></url>',
        '  <url><loc>https://fuziongaming.gg/news</loc></url>',
        '  <url><loc>https://fuziongaming.gg/roster</loc></url>',
        '  <url><loc>https://fuziongaming.gg/raid-progress</loc></url>',
        '  <url><loc>https://fuziongaming.gg/calendar</loc></url>',
        '  <url><loc>https://fuziongaming.gg/streams</loc></url>',
        '  <url><loc>https://fuziongaming.gg/applications</loc></url>',
        '  <url><loc>https://fuziongaming.gg/professions</loc></url>',
        '  <url><loc>https://fuziongaming.gg/charter</loc></url>',
        '  <url><loc>https://fuziongaming.gg/legal</loc></url>',
        '</urlset>',
        '',
      ].join('\n'),
    )
  })

  it('should leave out officer pages when building the list', () => {
    expect(buildSitemapXml('https://fuziongaming.gg')).not.toContain('/officer')
  })
})
