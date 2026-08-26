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

export interface GenerateResult {
  mode: GenerateMode
  jobDescription?: string
  selection: SelectionResult
  coverLetter?: CoverLetterResult
}

export interface GenerateRequest {
  jobDescription?: string
  jobDescriptionUrl?: string
}

