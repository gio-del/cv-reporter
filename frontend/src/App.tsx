import { BrowserRouter, Route, Routes } from 'react-router-dom'
import AppNav from '@/components/AppNav'
import AtsBrowsePage from '@/pages/AtsBrowsePage'
import EntriesListPage from '@/pages/EntriesListPage'
import EntryCreatePage from '@/pages/EntryCreatePage'
import EntryDetailPage from '@/pages/EntryDetailPage'
import GenerationPage from '@/pages/GenerationPage'
import JobListingCreatePage from '@/pages/JobListingCreatePage'
import JobListingsListPage from '@/pages/JobListingsListPage'
import ProfilePage from '@/pages/ProfilePage'
import SnippetCreatePage from '@/pages/SnippetCreatePage'
import SnippetDetailPage from '@/pages/SnippetDetailPage'
import SnippetsListPage from '@/pages/SnippetsListPage'
import StatsPage from '@/pages/StatsPage'
import TagLintPage from '@/pages/TagLintPage'
import { TooltipProvider } from '@/components/ui/tooltip'

function App() {
  return (
    <TooltipProvider>
      <BrowserRouter>
        <AppNav />
        <main className="mx-auto max-w-[960px] px-6 pt-6 pb-12 sm:px-4">
          <Routes>
            <Route path="/" element={<EntriesListPage />} />
            <Route path="/entries/new" element={<EntryCreatePage />} />
            <Route path="/tags" element={<TagLintPage />} />
            <Route path="/entries/*" element={<EntryDetailPage />} />
            <Route path="/profile" element={<ProfilePage />} />
            <Route path="/generate" element={<GenerationPage />} />
            <Route path="/jobs" element={<JobListingsListPage />} />
            <Route path="/stats" element={<StatsPage />} />
            <Route path="/jobs/new" element={<JobListingCreatePage />} />
            <Route path="/ats" element={<AtsBrowsePage />} />
            <Route path="/jobs/:id/generate" element={<GenerationPage />} />
            <Route path="/snippets" element={<SnippetsListPage />} />
            <Route path="/snippets/new" element={<SnippetCreatePage />} />
            <Route path="/snippets/:id" element={<SnippetDetailPage />} />
          </Routes>
        </main>
      </BrowserRouter>
    </TooltipProvider>
  )
}

export default App
