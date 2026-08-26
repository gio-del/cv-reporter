import { BrowserRouter, Route, Routes } from 'react-router-dom'
import EntriesListPage from './pages/EntriesListPage'
import EntryCreatePage from './pages/EntryCreatePage'
import EntryDetailPage from './pages/EntryDetailPage'
import GenerationPage from './pages/GenerationPage'
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
          <Route path="/generate" element={<GenerationPage />} />
        </Routes>
      </main>
    </BrowserRouter>
  )
}

export default App
