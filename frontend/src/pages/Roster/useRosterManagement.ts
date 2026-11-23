import { useMutation, useQueryClient } from '@tanstack/react-query'
import { setSpecialization } from '../../api/professions'
import { createCharacter, deleteCharacter, updateCharacter } from '../../api/roster'
import type { UpdateCharacterRequest } from '../../types/roster'
import type { SpecializationChange } from './characterForm'

interface UpdateVariables {
  id: string
  request: UpdateCharacterRequest
  specializations: SpecializationChange[]
}

export function useRosterManagement() {
  const queryClient = useQueryClient()
  const refetchRoster = () => queryClient.invalidateQueries({ queryKey: ['roster'] })

  const create = useMutation({ mutationFn: createCharacter, onSuccess: refetchRoster })
  const update = useMutation({
    mutationFn: async ({ id, request, specializations }: UpdateVariables) => {
      if (Object.keys(request).length > 0) await updateCharacter(id, request)
      for (const { professionId, specialization } of specializations) {
        await setSpecialization(professionId, specialization)
      }
    },
    // Also after a failure: the requests before the failing one were saved.
    onSettled: () =>
      Promise.all([refetchRoster(), queryClient.invalidateQueries({ queryKey: ['professions'] })]),
  })
  const remove = useMutation({ mutationFn: deleteCharacter, onSuccess: refetchRoster })

  return { create, update, remove }
}
