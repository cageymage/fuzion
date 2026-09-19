import { useQuery } from '@tanstack/react-query'
import { fetchCurrentUser } from '../api/auth'

export const currentUserQueryKey = ['auth', 'me'] as const

export function useCurrentUser() {
  return useQuery({
    queryKey: currentUserQueryKey,
    queryFn: fetchCurrentUser,
    staleTime: 5 * 60_000,
  })
}
