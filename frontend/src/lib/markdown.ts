const markdownImage = /!\[[^\]]*\]\(\s*([^)\s]+)/

export function firstMarkdownImageUrl(body: string): string | null {
  return markdownImage.exec(body)?.[1] ?? null
}
