import { useQuery } from '@tanstack/react-query'
import { fetchLatestNews } from '../../api/news'
import { fetchNextRaid } from '../../api/raids'
import { fetchLiveStreams, fetchSuggestedVideo } from '../../api/streams'

const homeNewsLimit = 3

export function useHomeData() {
  const news = useQuery({
    queryKey: ['news', 'latest', homeNewsLimit],
    queryFn: () => fetchLatestNews(homeNewsLimit),
  })
  const nextRaid = useQuery({ queryKey: ['raids', 'next'], queryFn: fetchNextRaid })
  const liveStreams = useQuery({ queryKey: ['streams', 'live'], queryFn: fetchLiveStreams })
  const nobodyIsLive = liveStreams.isSuccess && liveStreams.data.length === 0
  // The server picks the video at random, so refetching on window focus would swap
  // the video out from under a viewer who is watching it.
  const suggestedVideo = useQuery({
    queryKey: ['streams', 'suggested-video'],
    queryFn: fetchSuggestedVideo,
    enabled: nobodyIsLive,
    staleTime: Infinity,
    retry: false,
  })

  return { news, nextRaid, liveStreams, suggestedVideo }
}
