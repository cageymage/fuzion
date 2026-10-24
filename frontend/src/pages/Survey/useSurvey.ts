import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchSurvey, fetchSurveyResults, submitSurveyResponse } from '../../api/surveys'
import { ApiError } from '../../api/client'
import type { SubmitSurveyRequest } from '../../types/surveys'

const surveyQueryKey = (id: string) => ['survey', id] as const

function isNotFound(error: unknown): boolean {
  return error instanceof ApiError && error.status === 404
}

export function useSurvey(id: string) {
  return useQuery({
    queryKey: surveyQueryKey(id),
    queryFn: () => fetchSurvey(id),
    retry: (failureCount, error) => !isNotFound(error) && failureCount < 1,
  })
}

export function useSurveyResults(id: string) {
  return useQuery({
    queryKey: ['survey-results', id],
    queryFn: () => fetchSurveyResults(id),
  })
}

export function useSubmitSurveyResponse(id: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: SubmitSurveyRequest) => submitSurveyResponse(id, request),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: surveyQueryKey(id) }),
    // A 409 means the deadline passed while the form was open; refetching flips the page to the results.
    onError: (error) => {
      if (error instanceof ApiError && error.status === 409) {
        void queryClient.invalidateQueries({ queryKey: surveyQueryKey(id) })
      }
    },
  })
}
