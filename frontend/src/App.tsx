import { useEffect } from 'react'
import { BrowserRouter, Outlet, Route, Routes } from 'react-router-dom'
import { MotionConfig } from 'motion/react'
import { Toaster } from 'sonner'
import { AuthProvider } from './state/auth'
import { useTheme } from './state/stores'
import { TopNav } from './components/TopNav'
import { AuthDialog } from './components/AuthDialog'
import { Playground } from './pages/Playground'
import { Learn } from './pages/Learn'
import { Lesson } from './pages/Lesson'
import { NotFound, SharedQuery } from './pages/Misc'

function Shell() {
  return (
    <div className="app">
      <TopNav />
      <main className="app-main">
        <Outlet />
      </main>
      <AuthDialog />
    </div>
  )
}

export default function App() {
  const theme = useTheme((s) => s.theme)
  useEffect(() => {
    document.documentElement.dataset.theme = theme
    document.querySelector('meta[name="theme-color"]')?.setAttribute('content', theme === 'dark' ? '#0a0a0a' : '#fafaf7')
  }, [theme])

  return (
    // reducedMotion="user": springs and slides become instant for people who ask for less motion
    <MotionConfig reducedMotion="user">
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<Shell />}>
              <Route index element={<Playground />} />
              <Route path="learn" element={<Learn />} />
              <Route path="learn/:slug" element={<Lesson />} />
              <Route path="q/:id" element={<SharedQuery />} />
              <Route path="*" element={<NotFound />} />
            </Route>
          </Routes>
        </BrowserRouter>
        <Toaster
          theme={theme}
          position="bottom-right"
          toastOptions={{ className: 'toast' }}
          offset={20}
          mobileOffset={12}
        />
      </AuthProvider>
    </MotionConfig>
  )
}
