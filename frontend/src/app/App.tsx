import { Route, Routes } from 'react-router-dom'
import { AppFooter } from '../components/AppFooter/AppFooter'
import { AppHeader } from '../components/AppHeader/AppHeader'
import { Calendar } from '../pages/Calendar/Calendar'
import { ComingSoon } from '../pages/ComingSoon/ComingSoon'
import { Home } from '../pages/Home/Home'
import { News } from '../pages/News/News'
import { OfficerRaidProgress } from '../pages/OfficerRaidProgress/OfficerRaidProgress'
import { RaidProgress } from '../pages/RaidProgress/RaidProgress'
import { Roster } from '../pages/Roster/Roster'
import { Streams } from '../pages/Streams/Streams'
import styles from './App.module.css'
import { navItems } from './navigation'

export function App() {
  return (
    <div className={styles.page}>
      <div className={styles.glow} aria-hidden="true" />
      <AppHeader />
      <main className={styles.main}>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/news" element={<News />} />
          <Route path="/roster" element={<Roster />} />
          <Route path="/raid-progress" element={<RaidProgress />} />
          <Route path="/officer/raid-progress" element={<OfficerRaidProgress />} />
          <Route path="/calendar" element={<Calendar />} />
          <Route path="/streams" element={<Streams />} />
          {navItems
            .filter(
              (item) =>
                !['/', '/news', '/roster', '/raid-progress', '/calendar', '/streams'].includes(
                  item.path,
                ),
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
  )
}
