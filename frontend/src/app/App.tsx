import { Route, Routes } from 'react-router-dom'
import { AppFooter } from '../components/AppFooter/AppFooter'
import { AppHeader } from '../components/AppHeader/AppHeader'
import { ComingSoon } from '../pages/ComingSoon/ComingSoon'
import { Home } from '../pages/Home/Home'
import { News } from '../pages/News/News'
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
          {navItems
            .filter((item) => item.path !== '/' && item.path !== '/news')
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
