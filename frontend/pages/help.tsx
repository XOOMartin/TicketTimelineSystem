import { useState } from 'react'

import Badge from 'components/atoms/Badge'
import Message from 'components/atoms/Message'
import SearchBox from 'components/molecules/SearchBox'
import PageLayout from 'components/layouts/PageLayout'
import { PHASE_COLORS } from 'lib/colors'
import { GLOSSARY, Term } from 'lib/glossary'

import styles from 'styles/Help.module.css'

const matches = (t: Term, query: string) =>
  [t.term, t.aka ?? '', t.definition].some((text) => text.toLowerCase().includes(query))

// Definitions of the terms used across the app, grouped, with a search
export default function Help() {
  const [query, setQuery] = useState('')
  const q = query.trim().toLowerCase()
  const groups = GLOSSARY.map((g) => ({ ...g, terms: q ? g.terms.filter((t) => matches(t, q)) : g.terms })).filter(
    (g) => g.terms.length > 0,
  )

  return (
    <PageLayout title="Help" heading="Help" subheading="Definitions of the terms used in Ticket Timeline">
      <div className={styles.toolbar}>
        <SearchBox
          className={styles.search}
          value={query}
          onChange={setQuery}
          placeholder="Search terms"
          label="Search terms"
        />
        {!q && (
          <nav className={styles.jump} aria-label="Sections">
            {GLOSSARY.map((g) => (
              <a key={g.id} href={`#${g.id}`}>
                {g.title}
              </a>
            ))}
          </nav>
        )}
      </div>

      {groups.length === 0 && <Message>No terms match “{query.trim()}”.</Message>}

      {groups.map((g) => (
        <section key={g.id} id={g.id} className={styles.group}>
          <h2>{g.title}</h2>
          {g.intro && !q && <p className={styles.intro}>{g.intro}</p>}
          <dl className={styles.terms}>
            {g.terms.map((t) => (
              <div key={t.term} className={styles.term}>
                <dt>
                  {t.status ? (
                    <Badge value={t.status}>{t.term}</Badge>
                  ) : (
                    <span className={styles.name}>
                      {t.phase && <i className={styles.dot} style={{ background: PHASE_COLORS[t.phase] }} />}
                      {t.term}
                    </span>
                  )}
                  {t.aka && <span className={styles.aka}>{t.aka}</span>}
                </dt>
                <dd>{t.definition}</dd>
              </div>
            ))}
          </dl>
        </section>
      ))}
    </PageLayout>
  )
}
