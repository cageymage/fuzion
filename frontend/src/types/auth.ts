export interface AuthUser {
  id: string
  username: string
  avatarUrl: string | null
  isOfficer: boolean
  battlenetLinked: boolean
  battletag: string | null
}
