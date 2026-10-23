const ogStartMarker = '<!--og:start-->'
const ogEndMarker = '<!--og:end-->'

// Link previews (Discord, Bluesky) never run the app's JavaScript, so the About page gets its
// own HTML file at build time with these tags in place of the site-wide ones.
export function buildAboutPageHtml(indexHtml: string, siteBaseUrl: string): string {
  const start = indexHtml.indexOf(ogStartMarker)
  const end = indexHtml.indexOf(ogEndMarker)
  if (start < 0 || end < start) {
    throw new Error('index.html has no og block markers')
  }

  const url = `${siteBaseUrl}/about`
  const description =
    'Meet Fuzion, a PvE guild for World of Warcraft: Forever. A 20-person core team that teams up with partner guilds for 40-person raids. Apply to join.'
  const head = `
    <title>About | Fuzion</title>
    <meta name="description" content="${description}" />
    <link rel="canonical" href="${url}" />
    <meta property="og:type" content="website" />
    <meta property="og:site_name" content="Fuzion" />
    <meta property="og:title" content="About Fuzion" />
    <meta property="og:description" content="${description}" />
    <meta property="og:url" content="${url}" />
    <meta property="og:image" content="${siteBaseUrl}/about-og.jpg" />
    <meta property="og:image:width" content="1200" />
    <meta property="og:image:height" content="630" />
    <meta
      property="og:image:alt"
      content="A group of adventurers silhouetted in a snowy canyon, facing a glowing amber gateway in an ancient fortress."
    />
    <meta name="twitter:card" content="summary_large_image" />
    `
  return indexHtml.slice(0, start + ogStartMarker.length) + head + indexHtml.slice(end)
}
