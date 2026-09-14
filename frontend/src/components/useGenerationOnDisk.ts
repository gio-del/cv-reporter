import { useEffect, useState } from 'react'
import { generationFileOnDisk } from '@/api/client'

// Whether a Generation's Tailored CV is still on disk: null while checking
// (or with no slug), then true/false. A Generation whose files are gone is
// an expected state — output/ is derived and the user may clear it — so
// callers should show GENERATION_FILES_GONE_NOTE, not an error.
export function useGenerationOnDisk(slug: string | undefined): boolean | null {
  const [result, setResult] = useState<{ slug: string; onDisk: boolean } | null>(null)

  useEffect(() => {
    if (!slug) return
    let cancelled = false
    generationFileOnDisk(slug, 'cv.pdf').then((onDisk) => {
      if (!cancelled) setResult({ slug, onDisk })
    })
    return () => {
      cancelled = true
    }
  }, [slug])

  return slug && result?.slug === slug ? result.onDisk : null
}

export const GENERATION_FILES_GONE_NOTE =
  "This Generation's files are no longer on disk — output/ holds derived files that can be cleared at any time. " +
  'Its record (when, which Entries, language, usage) is intact; regenerate to get a fresh PDF.'
