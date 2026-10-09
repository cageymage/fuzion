import { lazy, Suspense } from 'react'
import { Route, Routes } from 'react-router-dom'
import { AppFooter } from '../components/AppFooter/AppFooter'
import { AppHeader } from '../components/AppHeader/AppHeader'
import { ToastProvider } from '../components/Toast/Toast'
import { Applications } from '../pages/Applications/Applications'
import { Calendar } from '../pages/Calendar/Calendar'
import { ComingSoon } from '../pages/ComingSoon/ComingSoon'
import { Home } from '../pages/Home/Home'
import { Legal } from '../pages/Legal/Legal'
import { News } from '../pages/News/News'
import { OfficerApplications } from '../pages/OfficerApplications/OfficerApplications'
import { OfficerNews } from '../pages/OfficerNews/OfficerNews'
import { OfficerRaidProgress } from '../pages/OfficerRaidProgress/OfficerRaidProgress'
import { Professions } from '../pages/Professions/Professions'
import { RaidProgress } from '../pages/RaidProgress/RaidProgress'
import { Roster } from '../pages/Roster/Roster'
import { Streams } from '../pages/Streams/Streams'
import styles from './App.module.css'
import { navItems } from './navigation'

// Loaded on demand: the Markdown renderer and editor are heavy and not needed on first paint.
const NewsPostPage = lazy(() =>
  import('../pages/NewsPost/NewsPost').then((module) => ({ default: module.NewsPostPage })),
)
const OfficerNewsEditor = lazy(() =>
  import('../pages/OfficerNewsEditor/OfficerNewsEditor').then((module) => ({
    default: module.OfficerNewsEditor,
  })),
)

export function App() {
  return (
    <ToastProvider>
      <div className={styles.page}>
        <div className={styles.glow} aria-hidden="true" />
        <AppHeader />
        <main className={styles.main}>
          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/news" element={<News />} />
            <Route
              path="/news/:id"
              element={
                <Suspense fallback={null}>
                  <NewsPostPage />
                </Suspense>
              }
            />
            <Route path="/roster" element={<Roster />} />
            <Route path="/raid-progress" element={<RaidProgress />} />
            <Route path="/officer/raid-progress" element={<OfficerRaidProgress />} />
            <Route path="/officer/applications" element={<OfficerApplications />} />
            <Route path="/officer/news" element={<OfficerNews />} />
            <Route
              path="/officer/news/:id"
              element={
                <Suspense fallback={null}>
                  <OfficerNewsEditor />
                </Suspense>
              }
            />
            <Route path="/calendar" element={<Calendar />} />
            <Route path="/streams" element={<Streams />} />
            <Route path="/applications" element={<Applications />} />
            <Route path="/professions" element={<Professions />} />
            <Route path="/legal" element={<Legal />} />
            {navItems
              .filter(
                (item) =>
                  ![
                    '/',
                    '/news',
                    '/roster',
                    '/raid-progress',
                    '/calendar',
                    '/streams',
                    '/applications',
                    '/professions',
                  ].includes(item.path),
              )
              .map((item) => (
                <Route
                  key={item.path}
                  path={item.path}
                  element={<ComingSoon title={item.label} />}
                />
              ))}
            <Route path="*" element={<ComingSoon title="Page not found" />} />
          </Routes>
        </main>
        <AppFooter />
      </div>
    </ToastProvider>
  )
}
