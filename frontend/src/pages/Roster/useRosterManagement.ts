import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createCharacter, deleteCharacter, updateCharacter } from '../../api/roster'
import type { UpdateCharacterRequest } from '../../types/roster'

export function useOfficerRoster() {
  const queryClient = useQueryClient()
  const refetchRoster = () => queryClient.invalidateQueries({ queryKey: ['roster'] })

  const create = useMutation({ mutationFn: createCharacter, onSuccess: refetchRoster })
  const update = useMutation({
    mutationFn: (variables: { id: string; request: UpdateCharacterRequest }) =>
      updateCharacter(variables.id, variables.request),
    onSuccess: refetchRoster,
  })
  const remove = useMutation({ mutationFn: deleteCharacter, onSuccess: refetchRoster })

  return { create, update, remove }
}
