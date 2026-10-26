import { Route, Routes, ScrollRestoration } from 'react-router-dom'
import { HubFooter } from '../components/HubFooter/HubFooter'
import { HubHeader } from '../components/HubHeader/HubHeader'
import { Home } from '../pages/Home/Home'
import { Legal } from '../pages/Legal/Legal'
import { NotFound } from '../pages/NotFound/NotFound'
import styles from './App.module.css'

export function App() {
  return (
    <div className={styles.page}>
      <div className={styles.glow} aria-hidden="true" />
      <HubHeader />
      <main className={styles.main}>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/legal" element={<Legal />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </main>
      <HubFooter />
      <ScrollRestoration />
    </div>
  )
}
