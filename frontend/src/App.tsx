import { BrowserRouter, Route, Routes } from 'react-router-dom'
import EntriesListPage from './pages/EntriesListPage'
import EntryCreatePage from './pages/EntryCreatePage'
import EntryDetailPage from './pages/EntryDetailPage'
import ProfilePage from './pages/ProfilePage'

function App() {
  return (
    <BrowserRouter>
      <main>
        <Routes>
          <Route path="/" element={<EntriesListPage />} />
          <Route path="/entries/new" element={<EntryCreatePage />} />
          <Route path="/entries/*" element={<EntryDetailPage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Routes>
      </main>
    </BrowserRouter>
  )
}

export default App
