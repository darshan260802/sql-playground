import { NavLink, useLocation } from 'react-router-dom'
import { motion } from 'motion/react'
import { Moon, Sun } from 'lucide-react'
import { useAuth } from '../state/auth'
import { useTheme, useUI } from '../state/stores'
import { springSnappy } from '../lib/util'
import { Logo } from './Logo'
import { AccountMenu } from './AccountMenu'

const links = [
  { to: '/', label: 'Playground' },
  { to: '/learn', label: 'Learn' },
]

export function TopNav() {
  const { user, ready } = useAuth()
  const { theme, toggle } = useTheme()
  const openAuth = useUI((s) => s.openAuth)
  const { pathname } = useLocation()
  const current = pathname.startsWith('/learn') ? '/learn' : '/'

  return (
    <header className="topnav">
      <NavLink to="/" className="topnav-brand" aria-label="SQL Playground home">
        <Logo />
      </NavLink>
      <nav className="topnav-links" aria-label="Main">
        {links.map((l) => (
          <NavLink key={l.to} to={l.to} className={`topnav-link ${current === l.to ? 'active' : ''}`} end={l.to === '/'}>
            {current === l.to && <motion.span layoutId="nav-pill" className="topnav-pill" transition={springSnappy} />}
            <span>{l.label}</span>
          </NavLink>
        ))}
      </nav>
      <div className="topnav-right">
        <button className="icon-btn" onClick={toggle} aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'} title="Toggle theme">
          {theme === 'dark' ? <Sun /> : <Moon />}
        </button>
        {ready &&
          (user ? (
            <AccountMenu />
          ) : (
            <>
              <button className="btn btn-ghost btn-sm hide-xs" onClick={() => openAuth('signin')}>
                Sign in
              </button>
              <button className="btn btn-primary btn-sm" onClick={() => openAuth('signup')}>
                Get started
              </button>
            </>
          ))}
      </div>
    </header>
  )
}
