import { ReactNode } from 'react'
import Link from 'next/link'

import Icon from 'components/atoms/Icon'
import LinkButton from 'components/atoms/LinkButton'

import styles from 'styles/ListCard.module.css'

export type ListCardItem = {
  key: string
  color: string
  primary: ReactNode
  secondary?: ReactNode
  aside?: ReactNode
  // Makes the whole row a link
  href?: string
}

type Props = {
  title: string
  items: ListCardItem[]
  emptyText: string
  // Rows past the limit are left out; the action leads to the full list
  limit?: number
  action?: { label: string; href: string }
}

// A short list of things that need a look, with a link to the full list.
export default function ListCard(props: Props) {
  const { title, items, emptyText, limit = 5, action } = props

  return (
    <section className={styles.card}>
      <header className={styles.header}>
        <h2>{title}</h2>
        <span className={styles.count}>{items.length}</span>
      </header>
      {items.length === 0 ? (
        <p className={styles.empty}>{emptyText}</p>
      ) : (
        <ul className={styles.list}>
          {items.slice(0, limit).map((item) => {
            const content = (
              <>
                <i className={styles.dot} style={{ background: item.color }} />
                <div className={styles.text}>
                  <span className={styles.primary}>{item.primary}</span>
                  {item.secondary && <span className={styles.secondary}>{item.secondary}</span>}
                </div>
                {item.aside && <span className={styles.aside}>{item.aside}</span>}
              </>
            )
            return item.href ? (
              <li key={item.key} className={styles.linked}>
                <Link href={item.href} className={`${styles.item} ${styles.link}`}>
                  {content}
                  <span className={styles.chevron}>
                    <Icon name="chevronRight" size={16} />
                  </span>
                </Link>
              </li>
            ) : (
              <li key={item.key} className={styles.item}>
                {content}
              </li>
            )
          })}
        </ul>
      )}
      {action && (
        <footer className={styles.footer}>
          <LinkButton href={action.href} variant="outline" size="sm">
            {action.label}
          </LinkButton>
        </footer>
      )}
    </section>
  )
}
