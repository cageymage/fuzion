import { useQuery } from '@tanstack/react-query'
import { fetchLiveStreams, fetchSuggestedVideo } from '../../api/streams'

export function useHomeData() {
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

  return { liveStreams, suggestedVideo, nobodyIsLive }
}
