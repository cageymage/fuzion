export interface AuthUser {
  id: string
  username: string
  avatarUrl: string | null
  isOfficer: boolean
  isGuildMember: boolean
  battlenetLinked: boolean
  battletag: string | null
}
