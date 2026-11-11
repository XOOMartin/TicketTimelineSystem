import { ReactNode, useEffect, useState } from 'react'
import Head from 'next/head'
import Link from 'next/link'
import { useRouter } from 'next/router'

import Avatar from 'components/atoms/Avatar'
import Button from 'components/atoms/Button'
import Icon, { IconName } from 'components/atoms/Icon'
import Message from 'components/atoms/Message'
import ThemeSwitch from 'components/molecules/ThemeSwitch'
import TimerPill from 'components/molecules/TimerPill'
import useAuth from 'hooks/useAuth'

import styles from 'styles/PageLayout.module.css'

type NavLink = { href: string; label: string; icon: IconName }

const ADMIN_LINKS: NavLink[] = [
  { href: '/admin', label: 'Dashboard', icon: 'dashboard' },
  { href: '/admin/members', label: 'Members', icon: 'users' },
  { href: '/admin/users', label: 'Users', icon: 'account' },
  { href: '/admin/reports', label: 'Reports', icon: 'report' },
]

const MEMBER_LINKS: NavLink[] = [
  { href: '/', label: 'My dashboard', icon: 'dashboard' },
  { href: '/timeline', label: 'My timeline', icon: 'ticket' },
  { href: '/checkins', label: 'My check-ins', icon: 'list' },
]

// For everyone, admins and members
const SUPPORT_LINKS: NavLink[] = [{ href: '/help', label: 'Help', icon: 'help' }]

type Props = {
  title: string
  heading: string
  subheading?: string
  // Buttons shown at the right of the page header
  actions?: ReactNode
  // Public pages (login, signup) skip the login check and the sidebar
  isPublic?: boolean
  requireAdmin?: boolean
  children: ReactNode
}

export default function PageLayout(props: Props) {
  const { title, heading, subheading, actions, isPublic = false, requireAdmin = false, children } = props
  const { pathname, replace } = useRouter()
  const { user, isLoading, logout } = useAuth()
  const [isNavOpen, setIsNavOpen] = useState(false)

  const mustLogIn = !isPublic && !isLoading && !user
  useEffect(() => {
    if (mustLogIn) replace('/login')
  }, [mustLogIn, replace])

  useEffect(() => {
    setIsNavOpen(false)
  }, [pathname])

  const head = (
    <Head>
      <title>{`${title} · Ticket Timeline`}</title>
    </Head>
  )

  if (isPublic) {
    return (
      <>
        {head}
        <main className={styles.auth}>
          <div className={styles.authCard}>
            <div className={styles.brand}>
              <span className={styles.logo}>TT</span>
              Ticket Timeline
            </div>
            <h1 className={styles.authHeading}>{heading}</h1>
            {subheading && <p className={styles.subheading}>{subheading}</p>}
            {children}
          </div>
        </main>
      </>
    )
  }

  let content = children
  if (!user) {
    content = <Message>Loading…</Message>
  } else if (requireAdmin && user.role !== 'admin') {
    content = <Message variant="error">This page is for admins only.</Message>
  }

  const isAdmin = user?.role === 'admin'

  const renderLinks = (links: NavLink[]) =>
    links.map((link) => (
      <Link
        key={link.href}
        href={link.href}
        className={link.href === pathname ? `${styles.navLink} ${styles.active}` : styles.navLink}
        aria-current={link.href === pathname ? 'page' : undefined}
      >
        <Icon name={link.icon} />
        {link.label}
      </Link>
    ))

  return (
    <>
      {head}
      <div className={styles.shell}>
        <aside className={isNavOpen ? `${styles.sidebar} ${styles.open}` : styles.sidebar}>
          <div className={styles.brand}>
            <span className={styles.logo}>TT</span>
            Ticket Timeline
            <Button
              variant="ghost"
              size="icon"
              className={styles.closeNav}
              onClick={() => setIsNavOpen(false)}
              aria-label="Close menu"
            >
              <Icon name="close" />
            </Button>
          </div>

          {user && (
            <div className={styles.profile}>
              <Avatar name={user.name} size={36} />
              <div className={styles.profileText}>
                <span className={styles.profileName}>{user.name}</span>
                <span className={styles.profileMeta}>
                  {user.team} · {isAdmin ? 'Admin' : 'Member'}
                </span>
              </div>
            </div>
          )}

          <nav className={styles.nav}>
            {isAdmin && (
              <>
                <span className={styles.navTitle}>Admin</span>
                {renderLinks(ADMIN_LINKS)}
              </>
            )}
            <span className={styles.navTitle}>My work</span>
            {renderLinks(MEMBER_LINKS)}
            <span className={styles.navTitle}>Support</span>
            {renderLinks(SUPPORT_LINKS)}
          </nav>

          <div className={styles.theme}>
            <ThemeSwitch />
          </div>

          {user && (
            <button type="button" className={styles.navLink} onClick={() => logout().then(() => replace('/login'))}>
              <Icon name="logout" />
              Log out
            </button>
          )}
        </aside>
        {isNavOpen && <div className={styles.scrim} onClick={() => setIsNavOpen(false)} />}

        <div className={styles.content}>
          <header className={styles.topbar}>
            <Button
              variant="ghost"
              size="icon"
              className={styles.menu}
              onClick={() => setIsNavOpen(true)}
              aria-label="Open menu"
            >
              <Icon name="menu" />
            </Button>
            <div className={styles.titles}>
              <h1>{heading}</h1>
              {subheading && <p className={styles.subheading}>{subheading}</p>}
            </div>
            {user && (
              <div className={styles.actions}>
                <TimerPill />
                {actions}
              </div>
            )}
          </header>
          <main className={styles.main}>{content}</main>
        </div>
      </div>
    </>
  )
}
