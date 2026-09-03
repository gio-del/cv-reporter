export type EntryType = 'experience' | 'project'

export interface Entry {
  id: string
  type: EntryType
  employer?: string
  client?: string
  role?: string
  name?: string
  location?: string
  start: string
  end: string | null
  flagship?: boolean
  tags: string[]
  repo?: string
  bullets?: string[]
}

export type EntryInput = Omit<Entry, 'id'>

export interface Education {
  degree: string
  institution: string
  program: string
  start: string
  end: string
  grade: string
  courses?: string[]
}

export interface Publication {
  title: string
  authors: string
  venue: string
  link?: string
  note?: string
}

export interface Award {
  title: string
  description?: string
}

export interface Activity {
  title: string
  description?: string
}

export interface Language {
  name: string
  level: string
}

export interface Profile {
  name: string
  location: string
  email: string
  phone: string
  linkedin: string
  github: string
  education: Education[]
  publications: Publication[]
  awards: Award[]
  activities: Activity[]
  languages: Language[]
}

export interface Snippet {
  id: string
  kind: string
  tags: string[]
  body: string
}

export type SnippetInput = Omit<Snippet, 'id'>

export interface SelectedBullet {
  sourceIndex: number
  source: string
  rewritten: string
}

export interface SelectedEntry {
  entryId: string
  reason: string
  bullets: SelectedBullet[]
}

export interface SelectionResult {
  entries: SelectedEntry[]
}

export type GenerateMode = 'default' | 'tailored'

export interface CoverLetterResult {
  body: string
  sourceSnippetIds?: string[]
}

export type RALSource = 'stated' | 'estimated' | 'n/a'

export interface RALRange {
  min?: number
  max?: number
  currency?: string
  source: RALSource
}

export interface GenerateResult {
  mode: GenerateMode
  jobDescription?: string
  selection: SelectionResult
  coverLetter?: CoverLetterResult
  ral?: RALRange
}

export interface GenerateRequest {
  jobDescription?: string
  jobDescriptionUrl?: string
}

export interface RenderRequest {
  slug: string
  selection: SelectionResult
  coverLetter?: { body: string }
}

export interface RenderResult {
  slug: string
  cvPath: string
  coverLetterPath?: string
  cvPageCount: number
}

export type JobListingSource = 'manual'

export interface JobListing {
  id: string
  company: string
  url?: string
  source: JobListingSource
  savedAt: string
  jobDescription: string
  ral: RALRange
}

export type ApplicationStatus = 'saved' | 'tailoring' | 'sent' | 'interviewing' | 'rejected' | 'offer'

export type ApplicationMethodKind = 'portal' | 'email' | 'easy_apply' | 'other'

export interface ApplicationMethod {
  kind: ApplicationMethodKind
  value?: string
}

export interface GenerationRecord {
  slug: string
  createdAt: string
  cvPath: string
  coverLetterPath?: string
}

export interface Contact {
  name: string
  email: string
}

export interface Application {
  id: string
  jobListingId: string
  status: ApplicationStatus
  method: ApplicationMethod
  contact?: Contact
  generations?: GenerationRecord[]
}

export interface RecordGenerationRequest {
  slug: string
  cvPath: string
  coverLetterPath?: string
}

export interface SaveJobListingRequest {
  company: string
  url?: string
  jobDescription?: string
  jobDescriptionUrl?: string
}

export interface JobListingWithApplication {
  jobListing: JobListing
  application: Application
}

export type SaveJobListingResult = JobListingWithApplication

export type AtsProvider = 'greenhouse' | 'lever' | 'ashby'

export interface AtsListing {
  title: string
  location: string
  url: string
  description: string
  alreadySaved: boolean
}

export interface TrackedBoard {
  id: string
  provider: AtsProvider
  slug: string
  label?: string
}

export interface AddTrackedBoardRequest {
  provider: AtsProvider
  slug: string
  label?: string
}

