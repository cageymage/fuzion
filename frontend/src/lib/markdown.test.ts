import { describe, expect, it } from 'vitest'
import { firstMarkdownImageUrl } from './markdown'

describe('firstMarkdownImageUrl', () => {
  it('should return the url of the first image when the body contains several', () => {
    const body = 'Intro\n\n![Kill](https://cdn.example/kill.png)\n\n![Group](https://cdn.example/group.png)'

    expect(firstMarkdownImageUrl(body)).toBe('https://cdn.example/kill.png')
  })

  it('should return an api image path when the image is uploaded to the site', () => {
    expect(firstMarkdownImageUrl('![](/api/images/abc-123)')).toBe('/api/images/abc-123')
  })

  it('should ignore the optional title after the url', () => {
    expect(firstMarkdownImageUrl('![Kill](https://cdn.example/kill.png "Final pull")')).toBe(
      'https://cdn.example/kill.png',
    )
  })

  it('should return null when the body has no image', () => {
    expect(firstMarkdownImageUrl('Just [a link](https://example.com) and text')).toBeNull()
  })

  it('should return null when the body is empty', () => {
    expect(firstMarkdownImageUrl('')).toBeNull()
  })
})
