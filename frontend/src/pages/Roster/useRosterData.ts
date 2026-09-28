import { useQuery } from '@tanstack/react-query'
import { fetchRoster } from '../../api/roster'

export function useRosterData() {
  return useQuery({ queryKey: ['roster'], queryFn: fetchRoster })
}
