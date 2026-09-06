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
