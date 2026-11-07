import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { fetchRecipes, type RecipeFilter } from '../../api/professions'

export function useRecipes(filter: RecipeFilter, enabled = true) {
  return useQuery({
    queryKey: ['recipes', filter.query.trim(), filter.characterId],
    queryFn: () => fetchRecipes(filter),
    enabled,
    placeholderData: keepPreviousData,
  })
}
