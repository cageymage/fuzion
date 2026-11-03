export interface SubmissionStatus {
  open: boolean
  attestationVersion: string
}

export interface Submitter {
  id: string
  username: string
}

export interface Submission {
  id: string
  submitter: Submitter
  description: string
  originalFilename: string
  contentType: string
  width: number
  height: number
  byteSize: number
  createdAt: string
  thumbUrl: string
  fullUrl: string
  downloadUrl: string
}

export interface SubmissionPage {
  total: number
  limit: number
  offset: number
  items: Submission[]
}

export interface SubmissionFileResult {
  filename: string
  submission?: Submission
  error?: string
}
