import { NavLink, useLocation } from 'react-router-dom'
import { motion } from 'motion/react'
import { BookOpen, LogIn, Moon, SquareTerminal, Sun } from 'lucide-react'
import { useAuth } from '../state/auth'
import { useTheme, useUI } from '../state/stores'
import { springSnappy } from '../lib/util'
import { useIsMobile } from '../lib/hooks'
import { LogoMark } from './Logo'
import { AccountMenu } from './AccountMenu'

const links = [
  { to: '/', label: 'Playground', icon: SquareTerminal },
  { to: '/learn', label: 'Learn', icon: BookOpen },
]

/** The destination the current page belongs to; none on pages outside both (a 404). */
function useSection() {
  const { pathname } = useLocation()
  if (pathname.startsWith('/learn')) return '/learn'
  if (pathname === '/' || pathname.startsWith('/q/')) return '/'
  return null
}

/** Desktop and tablet: an unlabelled icon rail with the product mark on top, utilities at the bottom. */
export function NavRail() {
  const { user, ready } = useAuth()
  const { theme, toggle } = useTheme()
  const openAuth = useUI((s) => s.openAuth)
  const current = useSection()

  return (
    <nav className="rail" aria-label="Main">
      <NavLink to="/" className="rail-mark" aria-label="SQL Playground">
        <LogoMark size={28} />
      </NavLink>
      <div className="rail-group">
        {links.map(({ to, label, icon: Icon }) => (
          <NavLink key={to} to={to} end={to === '/'} className="rail-item" aria-label={label} data-tip={label} aria-current={current === to ? 'page' : undefined}>
            <Icon />
            {current === to && <motion.span layoutId="rail-line" className="rail-line" transition={springSnappy} />}
          </NavLink>
        ))}
      </div>
      <div className="rail-group rail-bottom">
        <button className="rail-item" onClick={toggle} aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'} data-tip={theme === 'dark' ? 'Light theme' : 'Dark theme'}>
          {theme === 'dark' ? <Sun /> : <Moon />}
        </button>
        {ready &&
          (user ? (
            <AccountMenu placement="rail" />
          ) : (
            <button className="rail-item" onClick={() => openAuth('signin')} aria-label="Sign in" data-tip="Sign in">
              <LogIn />
            </button>
          ))}
      </div>
    </nav>
  )
}

/** Phones: the same destinations as a bottom bar, each with a short label. */
export function BottomBar() {
  const { user, ready } = useAuth()
  const { theme, toggle } = useTheme()
  const openAuth = useUI((s) => s.openAuth)
  const current = useSection()

  return (
    <nav className="bottombar" aria-label="Main">
      {links.map(({ to, label, icon: Icon }) => (
        <NavLink key={to} to={to} end={to === '/'} className="bottombar-item" aria-current={current === to ? 'page' : undefined}>
          {current === to && <motion.span layoutId="bottom-line" className="bottombar-line" transition={springSnappy} />}
          <Icon />
          <span>{label}</span>
        </NavLink>
      ))}
      <button className="bottombar-item" onClick={toggle} aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}>
        {theme === 'dark' ? <Sun /> : <Moon />}
        <span>{theme === 'dark' ? 'Light' : 'Dark'}</span>
      </button>
      {ready &&
        (user ? (
          <AccountMenu placement="bar" />
        ) : (
          <button className="bottombar-item" onClick={() => openAuth('signin')}>
            <LogIn />
            <span>Sign in</span>
          </button>
        ))}
    </nav>
  )
}

export function MainNav() {
  return useIsMobile() ? <BottomBar /> : <NavRail />
}
