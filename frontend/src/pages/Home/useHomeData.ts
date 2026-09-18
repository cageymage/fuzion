import { useQuery } from '@tanstack/react-query'
import { fetchNews } from '../../api/news'
import { fetchNextRaid } from '../../api/raids'
import { fetchLiveStreams } from '../../api/streams'

export function useHomeData() {
  const news = useQuery({ queryKey: ['news'], queryFn: fetchNews })
  const nextRaid = useQuery({ queryKey: ['raids', 'next'], queryFn: fetchNextRaid })
  const liveStreams = useQuery({ queryKey: ['streams', 'live'], queryFn: fetchLiveStreams })

  return { news, nextRaid, liveStreams }
}
