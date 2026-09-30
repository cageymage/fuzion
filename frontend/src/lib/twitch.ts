const TWITCH_HOSTS = new Set(['twitch.tv', 'www.twitch.tv'])
const LOGIN_PATTERN = /^\w+$/

export function twitchChannelFromUrl(channelUrl: string): string | null {
  let url: URL
  try {
    url = new URL(channelUrl)
  } catch {
    return null
  }
  if (!TWITCH_HOSTS.has(url.hostname)) return null
  const [login] = url.pathname.split('/').filter(Boolean)
  return login && LOGIN_PATTERN.test(login) ? login : null
}
