export type EntryType = 'experience' | 'project'

export interface EntryLastModified {
  at?: string
  subject?: string
}

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
  /** Absent when the Entry has no git history yet (freshly added, uncommitted). */
  lastModified?: EntryLastModified
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
  lastUsedAt?: string
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

export type RALSource = 'stated' | 'estimated' | 'n/a' | 'unresolved' | 'conflict'

export interface RALFigure {
  min: number
  max: number
  currency: string
}

export interface RALRange {
  min?: number
  max?: number
  currency?: string
  source: RALSource
  descriptionStated?: RALFigure
  listingStated?: RALFigure
}

export interface CallUsage {
  callType: string
  model: string
  inputTokens: number
  outputTokens: number
  cacheReadTokens?: number
  cacheWriteTokens?: number
  webSearchUses?: number
  estimatedCostUsd: number
}

export interface GenerationUsage {
  inputTokens: number
  outputTokens: number
  cacheReadTokens?: number
  cacheWriteTokens?: number
  webSearchUses?: number
  estimatedCostUsd: number
  calls?: CallUsage[]
}

export type GroundednessReason = 'no-source-match' | 'numeric-mismatch'

export interface GroundednessFlag {
  sentence: string
  reason: GroundednessReason
}

export interface BulletGroundedness {
  entryId: string
  sourceIndex: number
  flags: GroundednessFlag[]
}

export interface GroundednessResult {
  bullets?: BulletGroundedness[]
  coverLetter?: GroundednessFlag[]
}

export interface GenerateResult {
  mode: GenerateMode
  jobDescription?: string
  selection: SelectionResult
  coverLetter?: CoverLetterResult
  ral?: RALRange
  usage: GenerationUsage
  groundedness?: GroundednessResult
  language: string
}

export interface GenerateRequest {
  jobDescription?: string
  jobDescriptionUrl?: string
  languageOverride?: string
}

export interface RenderRequest {
  slug: string
  selection: SelectionResult
  coverLetter?: { body: string }
  language?: string
}

export type ParsabilityStatus = 'ok' | 'warning' | 'unavailable'

// ParsabilityResult is the ATS-parsability check's structured outcome
// (backend/internal/generation/parsability.go): non-blocking, surfaced at
// Visual Review as a warning badge alongside the CV/Cover Letter preview.
export interface ParsabilityResult {
  status: ParsabilityStatus
  missingFields?: string[]
  orderingViolations?: string[]
  reason?: string
}

export interface RenderResult {
  slug: string
  cvPath: string
  coverLetterPath?: string
  cvPageCount: number
  cvParsability: ParsabilityResult
  coverLetterParsability?: ParsabilityResult
}

// RALListQuery is GET /api/job-listings' optional RAL Range sort/filter
// query params (issue #51). sortOrder is only meaningful once a RAL sort
// is applied (asc/desc); ralCurrency defaults server-side to EUR when a
// min/max filter is set without one.
export interface RALListQuery {
  sortByRAL?: 'asc' | 'desc'
  ralMin?: number
  ralMax?: number
  ralCurrency?: string
}

export type JobListingSource = 'manual'

export interface JobListing {
  id: string
  title?: string
  company: string
  url?: string
  source: JobListingSource
  savedAt: string
  jobDescription: string
  ral: RALRange
  logo?: string
}

export type ApplicationStatus = 'saved' | 'tailoring' | 'sent' | 'interviewing' | 'rejected' | 'offer' | 'withdrawn'

export type ApplicationMethodKind = 'portal' | 'email' | 'easy_apply' | 'other' | 'unresolved'

export interface ApplicationMethod {
  kind: ApplicationMethodKind
  value?: string
}

export interface GenerationRecord {
  slug: string
  createdAt: string
  cvPath: string
  coverLetterPath?: string
  sourceSnippetIds?: string[]
  usage?: GenerationUsage
}

export interface Contact {
  name: string
  email: string
}

export interface Application {
  id: string
  jobListingId: string
  status: ApplicationStatus
  statusUpdatedAt?: string
  method: ApplicationMethod
  contact?: Contact
  isStale: boolean
  generations?: GenerationRecord[]
}

export interface StatusCount {
  status: ApplicationStatus
  count: number
}

export interface ConversionRate {
  from: ApplicationStatus
  to: ApplicationStatus
  rate: number
}

export interface StageTime {
  status: ApplicationStatus
  averageDays: number
  sampleSize: number
}

export interface ApplicationStats {
  total: number
  counts: StatusCount[]
  conversions: ConversionRate[]
  timeInStage: StageTime[]
}

export interface RecordGenerationRequest {
  slug: string
  cvPath: string
  coverLetterPath?: string
  sourceSnippetIds?: string[]
  usage?: GenerationUsage
  language?: string
  groundedness?: GroundednessResult
}

export interface SaveJobListingRequest {
  title?: string
  company: string
  url?: string
  jobDescription?: string
  jobDescriptionUrl?: string
  logoUrl?: string
}

export interface DuplicateMatch {
  jobListingId: string
  company: string
  title?: string
  savedAt: string
  score: number
}

export interface JobListingWithApplication {
  jobListing: JobListing
  application: Application
}

export type SaveJobListingResult = JobListingWithApplication & {
  duplicateWarning?: DuplicateMatch
}

export type AtsProvider = 'greenhouse' | 'lever' | 'ashby'

export interface AtsListing {
  title: string
  location: string
  url: string
  description: string
  alreadySaved: boolean
  logoUrl?: string
  new: boolean
}

export interface TrackedBoard {
  id: string
  provider: AtsProvider
  slug: string
  label?: string
  newCount: number
}

export interface AddTrackedBoardRequest {
  provider: AtsProvider
  slug: string
  label?: string
}

export type TagLintConfidence = 'confident' | 'suggested'

export interface TagLintOccurrence {
  tag: string
  entryId: string
  entryType: EntryType
}

export interface TagLintGroup {
  key: string
  confidence: TagLintConfidence
  occurrences: TagLintOccurrence[]
}

export interface TagLintReport {
  groups: TagLintGroup[]
  singletons: TagLintOccurrence[]
}

