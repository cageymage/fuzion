export type ApplicationRole = 'tank' | 'healer' | 'dps'

export interface SubmitApplicationRequest {
  applicantName: string
  characterName: string
  class: string
  role: ApplicationRole | ''
  availability: string
  discordHandle: string
  notes: string
}
