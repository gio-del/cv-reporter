import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
// Bundled brand faces — never a CDN (ADR-0004: the app is local-only and must
// work offline). Instrument Serif is display-only; Inter carries the UI.
import '@fontsource/inter/400.css'
import '@fontsource/inter/500.css'
import '@fontsource/inter/600.css'
import '@fontsource/instrument-serif/400.css'
import './index.css'
import App from './App.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
