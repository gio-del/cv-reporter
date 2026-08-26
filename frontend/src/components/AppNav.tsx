import { NavLink } from 'react-router-dom'

export default function AppNav() {
  return (
    <nav className="app-nav" aria-label="Primary">
      <strong>CV Reporter</strong>
      <NavLink to="/" end>
        Master Data
      </NavLink>
      <NavLink to="/profile">Profile</NavLink>
      <NavLink to="/snippets">Cover Letter Snippets</NavLink>
      <NavLink to="/generate">Generate</NavLink>
    </nav>
  )
}
