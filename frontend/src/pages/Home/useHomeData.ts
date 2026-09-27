import { useQuery } from '@tanstack/react-query'
import { fetchLatestNews } from '../../api/news'
import { fetchNextRaid } from '../../api/raids'
import { fetchLiveStreams } from '../../api/streams'

const homeNewsLimit = 3

export function useHomeData() {
  const news = useQuery({
    queryKey: ['news', 'latest', homeNewsLimit],
    queryFn: () => fetchLatestNews(homeNewsLimit),
  })
  const nextRaid = useQuery({ queryKey: ['raids', 'next'], queryFn: fetchNextRaid })
  const liveStreams = useQuery({ queryKey: ['streams', 'live'], queryFn: fetchLiveStreams })

  return { news, nextRaid, liveStreams }
}
