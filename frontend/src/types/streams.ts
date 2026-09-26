export interface Stream {
  id: string
  streamerName: string
  gameName: string
  viewerCount: number
  thumbnailUrl: string | null
  channelUrl: string
  isLive: boolean
}
