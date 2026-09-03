import { NavLink } from 'react-router-dom'
import { cn } from '@/lib/utils'

const navItems = [
  { to: '/', label: 'Master Data', end: true },
  { to: '/profile', label: 'Profile', end: false },
  { to: '/snippets', label: 'Cover Letter Snippets', end: false },
  { to: '/generate', label: 'Generate', end: false },
  { to: '/jobs', label: 'Job Listings', end: false },
  { to: '/ats', label: 'Browse ATS Boards', end: false },
]

export default function AppNav() {
  return (
    <nav
      className="flex flex-wrap items-center gap-x-6 gap-y-2 border-b border-border bg-card px-6 py-4"
      aria-label="Primary"
    >
      <strong className="mr-auto font-semibold">CV Reporter</strong>
      {navItems.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          end={item.end}
          className={({ isActive }) =>
            cn(
              'border-b-2 border-transparent py-1 font-medium text-muted-foreground no-underline hover:text-foreground',
              isActive && 'border-primary text-primary',
            )
          }
        >
          {item.label}
        </NavLink>
      ))}
    </nav>
  )
}
