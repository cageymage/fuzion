import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError } from '../../api/client'
import { fetchRankRequests, resolveRankRequest } from '../../api/rankRequests'
import type { RankRequestDecision } from '../../types/rankRequests'

const queryKey = ['rank-requests'] as const

function isAccessDenied(error: unknown): boolean {
  return error instanceof ApiError && (error.status === 401 || error.status === 403)
}

export function useRankRequestList() {
  return useQuery({
    queryKey: [...queryKey, 'list'],
    queryFn: fetchRankRequests,
    // Retrying a 401 or 403 only delays the not-authorized message for someone following the Discord link.
    retry: (failureCount, error) => !isAccessDenied(error) && failureCount < 1,
  })
}

export function useResolveRankRequest() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, status, note }: { id: string; status: RankRequestDecision; note: string }) =>
      resolveRankRequest(id, status, note),
    // Also refreshes the Dashboard's pending count, which lives under the same key.
    onSuccess: () => queryClient.invalidateQueries({ queryKey }),
  })
}
