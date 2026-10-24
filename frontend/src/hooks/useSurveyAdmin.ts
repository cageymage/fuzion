import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  addSurveyOption,
  createSurvey,
  fetchLiveSurveyResults,
  fetchSurveyResponses,
  updateSurvey,
  voidSurveyResponse,
} from '../api/surveys'
import { surveyQueryKey } from '../pages/Survey/useSurvey'
import type { CreateSurveyRequest, UpdateSurveyRequest, VoidSurveyResponseRequest } from '../types/surveys'

const responsesQueryKey = (id: string) => ['survey-responses', id] as const
const liveResultsQueryKey = (id: string) => ['survey-live-results', id] as const

const openSurveyRefreshMs = 30_000

export function useLiveSurveyResults(id: string) {
  return useQuery({
    queryKey: liveResultsQueryKey(id),
    queryFn: () => fetchLiveSurveyResults(id),
    refetchInterval: openSurveyRefreshMs,
  })
}

export function useSurveyResponses(id: string, { refresh }: { refresh: boolean }) {
  return useQuery({
    queryKey: responsesQueryKey(id),
    queryFn: () => fetchSurveyResponses(id),
    refetchInterval: refresh ? openSurveyRefreshMs : false,
  })
}

export function useCreateSurvey() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: CreateSurveyRequest) => createSurvey(request),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['surveys'] }),
  })
}

export function useUpdateSurvey(id: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: UpdateSurveyRequest) => updateSurvey(id, request),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: surveyQueryKey(id) }),
        queryClient.invalidateQueries({ queryKey: ['surveys'] }),
        queryClient.invalidateQueries({ queryKey: ['survey-results', id] }),
      ]),
  })
}

export function useAddSurveyOption(id: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (variables: { questionId: string; option: string }) =>
      addSurveyOption(id, variables.questionId, variables.option),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: surveyQueryKey(id) }),
        queryClient.invalidateQueries({ queryKey: liveResultsQueryKey(id) }),
        queryClient.invalidateQueries({ queryKey: ['survey-results', id] }),
      ]),
  })
}

export function useVoidSurveyResponse(surveyId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (variables: { responseId: string } & VoidSurveyResponseRequest) => {
      const { responseId, ...request } = variables
      return voidSurveyResponse(surveyId, responseId, request)
    },
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: responsesQueryKey(surveyId) }),
        queryClient.invalidateQueries({ queryKey: liveResultsQueryKey(surveyId) }),
        queryClient.invalidateQueries({ queryKey: ['survey-results', surveyId] }),
      ]),
  })
}
