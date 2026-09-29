import { useQuery } from '@tanstack/react-query'
import { fetchRaidProgress, fetchRaidTiers } from '../../api/raidProgress'

export function useRaidProgressData() {
  return useQuery({ queryKey: ['raid-progress'], queryFn: fetchRaidProgress })
}

export function useRaidTiersData() {
  return useQuery({ queryKey: ['raid-tiers'], queryFn: fetchRaidTiers })
}
