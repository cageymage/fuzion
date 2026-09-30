import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  addRaidBoss,
  createRaidTier,
  deleteRaidBoss,
  deleteRaidTier,
  renameRaidBoss,
  renameRaidTier,
  reorderRaidBosses,
  reorderRaidTiers,
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
  const renameTier = useMutation({
    mutationFn: (variables: { id: string; name: string }) => renameRaidTier(variables.id, variables.name),
    onSuccess: refetchProgress,
  })
  const reorderTiers = useMutation({ mutationFn: reorderRaidTiers, onSuccess: refetchProgress })

  const setBossKilled = useMutation({
    mutationFn: (variables: { id: string; killed: boolean }) =>
      setRaidBossKilled(variables.id, variables.killed),
    onSuccess: refetchProgress,
  })
  const addBoss = useMutation({
    mutationFn: (variables: { tierId: string; name: string }) => addRaidBoss(variables.tierId, variables.name),
    onSuccess: refetchProgress,
  })
  const renameBoss = useMutation({
    mutationFn: (variables: { id: string; name: string }) => renameRaidBoss(variables.id, variables.name),
    onSuccess: refetchProgress,
  })
  const reorderBosses = useMutation({
    mutationFn: (variables: { tierId: string; ids: string[] }) => reorderRaidBosses(variables.tierId, variables.ids),
    onSuccess: refetchProgress,
  })

  const deleteTier = useMutation({ mutationFn: deleteRaidTier, onSuccess: refetchProgress })
  const deleteBoss = useMutation({ mutationFn: deleteRaidBoss, onSuccess: refetchProgress })

  return {
    createTier,
    setTierCurrent,
    renameTier,
    reorderTiers,
    setBossKilled,
    addBoss,
    renameBoss,
    reorderBosses,
    deleteTier,
    deleteBoss,
  }
}
