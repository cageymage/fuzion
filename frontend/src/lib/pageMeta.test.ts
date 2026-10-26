import { describe, expect, it } from 'vitest'
import { buildAboutPageHtml } from './pageMeta'

const shell =
  '<head><meta charset="utf-8" /><!--og:start--><title>Fuzion</title><meta property="og:title" content="Fuzion" /><!--og:end--><link rel="icon" href="/favicon.ico" /></head>'

describe('buildAboutPageHtml', () => {
  it('should replace the site-wide tags with the about page tags when the shell has og markers', () => {
    const html = buildAboutPageHtml(shell, 'https://forever.fuziongaming.gg')

    expect(html).toContain('<title>About | Fuzion</title>')
    expect(html).toContain('<meta property="og:title" content="About Fuzion" />')
    expect(html).not.toContain('<title>Fuzion</title>')
  })

  it('should point the canonical link, og url and og image at absolute about urls when given a base url', () => {
    const html = buildAboutPageHtml(shell, 'https://forever.fuziongaming.gg')

    expect(html).toContain('<link rel="canonical" href="https://forever.fuziongaming.gg/about" />')
    expect(html).toContain('<meta property="og:url" content="https://forever.fuziongaming.gg/about" />')
    expect(html).toContain(
      '<meta property="og:image" content="https://forever.fuziongaming.gg/about-og.jpg" />',
    )
  })

  it('should keep everything outside the og markers when replacing the tags', () => {
    const html = buildAboutPageHtml(shell, 'https://forever.fuziongaming.gg')

    expect(html.startsWith('<head><meta charset="utf-8" /><!--og:start-->')).toBe(true)
    expect(html.endsWith('<!--og:end--><link rel="icon" href="/favicon.ico" /></head>')).toBe(true)
  })

  it('should throw when the shell has no og markers', () => {
    expect(() => buildAboutPageHtml('<head></head>', 'https://forever.fuziongaming.gg')).toThrow(
      'index.html has no og block markers',
    )
  })
})
