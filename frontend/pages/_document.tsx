import { Head, Html, Main, NextScript } from 'next/document'

import { THEME_SCRIPT } from 'lib/theme'

export default function Document() {
  return (
    <Html lang="en">
      <Head>
        {/* Picks light or dark before the first paint */}
        <script dangerouslySetInnerHTML={{ __html: THEME_SCRIPT }} />
      </Head>
      <body>
        <Main />
        <NextScript />
      </body>
    </Html>
  )
}
