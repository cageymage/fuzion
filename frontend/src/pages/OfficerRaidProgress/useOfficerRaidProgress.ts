import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  createRaidTier,
  deleteRaidBoss,
  deleteRaidTier,
  setRaidBossKilled,
  setRaidTierCurrent,
} from '../../api/raidProgress'

export function useOfficerRaidProgress() {
  const queryClient = useQueryClient()
  const refetchProgress = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: ['raid-tiers'] }),
      queryClient.invalidateQueries({ queryKey: ['raid-progress'] }),
    ])

  const createTier = useMutation({ mutationFn: createRaidTier, onSuccess: refetchProgress })
  const setTierCurrent = useMutation({
    mutationFn: (variables: { id: string; isCurrent: boolean }) =>
      setRaidTierCurrent(variables.id, variables.isCurrent),
    onSuccess: refetchProgress,
  })
  const setBossKilled = useMutation({
    mutationFn: (variables: { id: string; killed: boolean }) =>
      setRaidBossKilled(variables.id, variables.killed),
    onSuccess: refetchProgress,
  })

  const deleteTier = useMutation({ mutationFn: deleteRaidTier, onSuccess: refetchProgress })
  const deleteBoss = useMutation({ mutationFn: deleteRaidBoss, onSuccess: refetchProgress })

  return { createTier, setTierCurrent, setBossKilled, deleteTier, deleteBoss }
}
