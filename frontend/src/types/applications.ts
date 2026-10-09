export type ApplicationRole = 'tank' | 'healer' | 'dps'

export interface SubmitApplicationRequest {
  applicantName: string
  characterName: string
  class: string
  role: ApplicationRole | ''
  availability: string
  discordHandle: string
  notes: string
  turnstileToken: string
}

export type ApplicationStatus = 'pending' | 'accepted' | 'declined'

export type ApplicationStatusFilter = ApplicationStatus | 'all'

export type ReviewDecision = Exclude<ApplicationStatus, 'pending'>

export interface Application {
  id: string
  applicantName: string
  characterName: string
  class: string
  role: ApplicationRole
  availability: string
  discordHandle: string
  notes: string
  status: ApplicationStatus
  submittedAt: string
  reviewedBy: string | null
  reviewedAt: string | null
  reviewNote: string
}

export interface ReviewApplicationRequest {
  status: ReviewDecision
  reviewNote?: string
}
