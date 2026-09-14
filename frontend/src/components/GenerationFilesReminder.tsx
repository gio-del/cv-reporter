import { generationFileUrl } from '@/api/client'
import type { GenerationRecord } from '@/api/types'
import { GENERATION_FILES_GONE_NOTE, useGenerationOnDisk } from '@/components/useGenerationOnDisk'

// Points at the latest Generation's Tailored CV/Cover Letter, with an
// explicit reminder to attach them — mailto: links (and manual portal
// uploads) can't carry attachments, so the user must never forget (stories
// 8, 10).
export default function GenerationFilesReminder({ generations }: { generations: GenerationRecord[] | undefined }) {
  const latest = generations && generations.length > 0 ? generations[generations.length - 1] : null
  const onDisk = useGenerationOnDisk(latest?.slug)

  if (!latest) {
    return (
      <p className="mb-0 text-sm text-muted-foreground">
        No Tailored CV generated yet for this Application — generate one before applying.
      </p>
    )
  }

  // output/ is derived (ADR-0008): missing files are expected, not an error.
  if (onDisk === false) {
    return <p className="mb-0 text-sm text-muted-foreground">{GENERATION_FILES_GONE_NOTE}</p>
  }

  return (
    <p className="mb-0 text-sm">
      Remember to attach your Tailored CV{latest.coverLetterPath && ' and Cover Letter'} — this can't carry
      attachments for you:{' '}
      <a href={generationFileUrl(latest.slug, 'cv.pdf')} target="_blank" rel="noreferrer">
        CV
      </a>
      {latest.coverLetterPath && (
        <>
          {' · '}
          <a href={generationFileUrl(latest.slug, 'cover-letter.pdf')} target="_blank" rel="noreferrer">
            Cover Letter
          </a>
        </>
      )}
    </p>
  )
}
