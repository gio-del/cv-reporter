import { BrowserRouter, Route, Routes } from 'react-router-dom'
import AppNav from './components/AppNav'
import EntriesListPage from './pages/EntriesListPage'
import EntryCreatePage from './pages/EntryCreatePage'
import EntryDetailPage from './pages/EntryDetailPage'
import GenerationPage from './pages/GenerationPage'
import ProfilePage from './pages/ProfilePage'
import SnippetCreatePage from './pages/SnippetCreatePage'
import SnippetDetailPage from './pages/SnippetDetailPage'
import SnippetsListPage from './pages/SnippetsListPage'

function App() {
  return (
    <BrowserRouter>
      <AppNav />
      <main>
        <Routes>
          <Route path="/" element={<EntriesListPage />} />
          <Route path="/entries/new" element={<EntryCreatePage />} />
          <Route path="/entries/*" element={<EntryDetailPage />} />
          <Route path="/profile" element={<ProfilePage />} />
          <Route path="/generate" element={<GenerationPage />} />
          <Route path="/snippets" element={<SnippetsListPage />} />
          <Route path="/snippets/new" element={<SnippetCreatePage />} />
          <Route path="/snippets/:id" element={<SnippetDetailPage />} />
        </Routes>
      </main>
    </BrowserRouter>
  )
}

export default App
