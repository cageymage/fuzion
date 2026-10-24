import { useLocation } from 'react-router-dom'
import { loginUrlFor } from '../api/auth'

export function useLoginUrl(): string {
  const { pathname, search } = useLocation()
  return loginUrlFor(pathname + search)
}
