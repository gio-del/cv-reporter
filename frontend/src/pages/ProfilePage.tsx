import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getProfile, updateProfile } from '../api/client'
import type { Activity, Award, Education, Language, Profile, Publication } from '../api/types'

function emptyEducation(): Education {
  return { degree: '', institution: '', program: '', start: '', end: '', grade: '', courses: [] }
}
function emptyPublication(): Publication {
  return { title: '', authors: '', venue: '', link: '', note: '' }
}
function emptyAward(): Award {
  return { title: '', description: '' }
}
function emptyActivity(): Activity {
  return { title: '', description: '' }
}
function emptyLanguage(): Language {
  return { name: '', level: '' }
}

function ListSection<T>({
  title,
  items,
  onChange,
  makeEmpty,
  renderItem,
}: {
  title: string
  items: T[]
  onChange: (items: T[]) => void
  makeEmpty: () => T
  renderItem: (item: T, update: (item: T) => void) => React.ReactNode
}) {
  return (
    <fieldset>
      <legend>{title}</legend>
      {items.map((item, i) => (
        <div key={i}>
          {renderItem(item, (updated) => {
            const next = [...items]
            next[i] = updated
            onChange(next)
          })}
          <button type="button" onClick={() => onChange(items.filter((_, j) => j !== i))}>
            Remove
          </button>
          <hr />
        </div>
      ))}
      <button type="button" onClick={() => onChange([...items, makeEmpty()])}>
        + Add {title.slice(0, -1)}
      </button>
    </fieldset>
  )
}

function ProfileEditForm({ profile, onSaved, onCancel }: { profile: Profile; onSaved: (p: Profile) => void; onCancel: () => void }) {
  const [form, setForm] = useState<Profile>(profile)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  function set<K extends keyof Profile>(key: K, value: Profile[K]) {
    setForm((f) => ({ ...f, [key]: value }))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      const saved = await updateProfile(form)
      onSaved(saved)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      {error && <p role="alert">{error}</p>}

      <fieldset>
        <legend>Contact Info</legend>
        <label>
          Name
          <input value={form.name} onChange={(e) => set('name', e.target.value)} />
        </label>
        <label>
          Location
          <input value={form.location} onChange={(e) => set('location', e.target.value)} />
        </label>
        <label>
          Email
          <input value={form.email} onChange={(e) => set('email', e.target.value)} />
        </label>
        <label>
          Phone
          <input value={form.phone} onChange={(e) => set('phone', e.target.value)} />
        </label>
        <label>
          LinkedIn
          <input value={form.linkedin} onChange={(e) => set('linkedin', e.target.value)} />
        </label>
        <label>
          GitHub
          <input value={form.github} onChange={(e) => set('github', e.target.value)} />
        </label>
      </fieldset>

      <ListSection
        title="Education"
        items={form.education}
        onChange={(items) => set('education', items)}
        makeEmpty={emptyEducation}
        renderItem={(item, update) => (
          <>
            <label>
              Degree
              <input value={item.degree} onChange={(e) => update({ ...item, degree: e.target.value })} />
            </label>
            <label>
              Institution
              <input value={item.institution} onChange={(e) => update({ ...item, institution: e.target.value })} />
            </label>
            <label>
              Program
              <input value={item.program} onChange={(e) => update({ ...item, program: e.target.value })} />
            </label>
            <label>
              Start
              <input value={item.start} onChange={(e) => update({ ...item, start: e.target.value })} />
            </label>
            <label>
              End
              <input value={item.end} onChange={(e) => update({ ...item, end: e.target.value })} />
            </label>
            <label>
              Grade
              <input value={item.grade} onChange={(e) => update({ ...item, grade: e.target.value })} />
            </label>
            <label>
              Courses (comma-separated)
              <input
                value={(item.courses ?? []).join(', ')}
                onChange={(e) =>
                  update({
                    ...item,
                    courses: e.target.value
                      .split(',')
                      .map((c) => c.trim())
                      .filter(Boolean),
                  })
                }
              />
            </label>
          </>
        )}
      />

      <ListSection
        title="Publications"
        items={form.publications}
        onChange={(items) => set('publications', items)}
        makeEmpty={emptyPublication}
        renderItem={(item, update) => (
          <>
            <label>
              Title
              <input value={item.title} onChange={(e) => update({ ...item, title: e.target.value })} />
            </label>
            <label>
              Authors
              <input value={item.authors} onChange={(e) => update({ ...item, authors: e.target.value })} />
            </label>
            <label>
              Venue
              <input value={item.venue} onChange={(e) => update({ ...item, venue: e.target.value })} />
            </label>
            <label>
              Link
              <input value={item.link ?? ''} onChange={(e) => update({ ...item, link: e.target.value })} />
            </label>
            <label>
              Note
              <textarea value={item.note ?? ''} onChange={(e) => update({ ...item, note: e.target.value })} />
            </label>
          </>
        )}
      />

      <ListSection
        title="Awards"
        items={form.awards}
        onChange={(items) => set('awards', items)}
        makeEmpty={emptyAward}
        renderItem={(item, update) => (
          <>
            <label>
              Title
              <input value={item.title} onChange={(e) => update({ ...item, title: e.target.value })} />
            </label>
            <label>
              Description
              <input value={item.description ?? ''} onChange={(e) => update({ ...item, description: e.target.value })} />
            </label>
          </>
        )}
      />

      <ListSection
        title="Activities"
        items={form.activities}
        onChange={(items) => set('activities', items)}
        makeEmpty={emptyActivity}
        renderItem={(item, update) => (
          <>
            <label>
              Title
              <input value={item.title} onChange={(e) => update({ ...item, title: e.target.value })} />
            </label>
            <label>
              Description
              <input value={item.description ?? ''} onChange={(e) => update({ ...item, description: e.target.value })} />
            </label>
          </>
        )}
      />

      <ListSection
        title="Languages"
        items={form.languages}
        onChange={(items) => set('languages', items)}
        makeEmpty={emptyLanguage}
        renderItem={(item, update) => (
          <>
            <label>
              Name
              <input value={item.name} onChange={(e) => update({ ...item, name: e.target.value })} />
            </label>
            <label>
              Level
              <input value={item.level} onChange={(e) => update({ ...item, level: e.target.value })} />
            </label>
          </>
        )}
      />

      <div className="form-actions">
        <button type="submit" disabled={saving}>
          {saving ? 'Saving…' : 'Save'}
        </button>
        <button type="button" onClick={onCancel} disabled={saving}>
          Cancel
        </button>
      </div>
    </form>
  )
}

export default function ProfilePage() {
  const [profile, setProfile] = useState<Profile | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState(false)

  useEffect(() => {
    getProfile()
      .then(setProfile)
      .catch((e) => setError(e.message))
  }, [])

  if (error) return <p role="alert">{error}</p>
  if (!profile) return <p>Loading…</p>

  if (editing) {
    return (
      <>
        <p className="breadcrumb">
          <Link to="/">← Back to Master Data</Link>
        </p>
        <h1>Edit Profile</h1>
        <ProfileEditForm
          profile={profile}
          onSaved={(saved) => {
            setProfile(saved)
            setEditing(false)
          }}
          onCancel={() => setEditing(false)}
        />
      </>
    )
  }

  return (
    <>
      <p>
        <Link to="/">← Back to Master Data</Link>
      </p>
      <h1>Profile</h1>

      <section>
        <h2>Contact Info</h2>
        <dl>
          <dt>Name</dt>
          <dd>{profile.name}</dd>
          <dt>Location</dt>
          <dd>{profile.location}</dd>
          <dt>Email</dt>
          <dd>{profile.email}</dd>
          <dt>Phone</dt>
          <dd>{profile.phone}</dd>
          <dt>LinkedIn</dt>
          <dd>{profile.linkedin}</dd>
          <dt>GitHub</dt>
          <dd>{profile.github}</dd>
        </dl>
      </section>

      <section>
        <h2>Education</h2>
        <ul>
          {profile.education.map((e, i) => (
            <li key={i}>
              {e.degree}, {e.institution} — {e.program} ({e.start}–{e.end})
            </li>
          ))}
        </ul>
      </section>

      <section>
        <h2>Publications</h2>
        <ul>
          {profile.publications.map((p, i) => (
            <li key={i}>{p.title}</li>
          ))}
        </ul>
      </section>

      <section>
        <h2>Awards</h2>
        <ul>
          {profile.awards.map((a, i) => (
            <li key={i}>{a.title}</li>
          ))}
        </ul>
      </section>

      <section>
        <h2>Activities</h2>
        <ul>
          {profile.activities.map((a, i) => (
            <li key={i}>{a.title}</li>
          ))}
        </ul>
      </section>

      <section>
        <h2>Languages</h2>
        <ul>
          {profile.languages.map((l, i) => (
            <li key={i}>
              {l.name} — {l.level}
            </li>
          ))}
        </ul>
      </section>

      <div className="form-actions">
        <button type="button" onClick={() => setEditing(true)}>
          Edit
        </button>
      </div>
    </>
  )
}
