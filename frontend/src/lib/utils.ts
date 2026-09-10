import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// "Job Title — Company" everywhere a Job Listing is displayed, or just
// Company when no Title was captured/entered (CONTEXT.md's Job Title entry).
export function jobListingHeading({ title, company }: { title?: string; company: string }): string {
  return title ? `${title} — ${company}` : company
}

const RELATIVE_TIME_UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ['year', 60 * 60 * 24 * 365],
  ['month', 60 * 60 * 24 * 30],
  ['week', 60 * 60 * 24 * 7],
  ['day', 60 * 60 * 24],
  ['hour', 60 * 60],
  ['minute', 60],
]

const relativeTimeFormatter = new Intl.RelativeTimeFormat('en', { numeric: 'auto' })

// Formats an ISO timestamp as "3 months ago", for an Entry's last-modified
// date (see the Entry change history PRD). Falls back to "just now" for
// anything under a minute.
export function formatRelativeTime(iso: string): string {
  const seconds = (Date.now() - new Date(iso).getTime()) / 1000
  for (const [unit, unitSeconds] of RELATIVE_TIME_UNITS) {
    if (seconds >= unitSeconds) {
      return relativeTimeFormatter.format(-Math.round(seconds / unitSeconds), unit)
    }
  }
  return 'just now'
}
