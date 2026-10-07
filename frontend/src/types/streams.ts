export interface SuggestedVideo {
  id: string
  title: string
}

export interface Stream {
  id: string
  streamerName: string
  gameName: string
  title: string
  viewerCount: number
  thumbnailUrl: string | null
  avatarUrl: string | null
  channelUrl: string
  isLive: boolean
  isLiveOtherGame: boolean
}
