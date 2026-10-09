import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchApplications, reviewApplication } from '../../api/applications'
import { ApiError } from '../../api/client'
import type { ApplicationStatusFilter, ReviewApplicationRequest } from '../../types/applications'

const queryKey = ['officer-applications'] as const

function isAccessDenied(error: unknown): boolean {
  return error instanceof ApiError && (error.status === 401 || error.status === 403)
}

export function useApplicationList(status: ApplicationStatusFilter) {
  return useQuery({
    queryKey: [...queryKey, status],
    queryFn: () => fetchApplications(status),
    // Retrying a 401 or 403 only delays the not-authorized message for someone following the Discord link.
    retry: (failureCount, error) => !isAccessDenied(error) && failureCount < 1,
  })
}

export function useReviewApplication() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (variables: { id: string } & ReviewApplicationRequest) => {
      const { id, ...request } = variables
      return reviewApplication(id, request)
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey }),
  })
}
