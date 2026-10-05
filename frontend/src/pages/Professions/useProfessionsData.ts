import { useQuery } from '@tanstack/react-query'
import { fetchProfessions } from '../../api/professions'

export function useProfessionsData() {
  return useQuery({ queryKey: ['professions'], queryFn: fetchProfessions })
}
