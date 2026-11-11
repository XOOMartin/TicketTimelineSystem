import type { AppProps } from 'next/app'
import { Poppins } from 'next/font/google'

import { AuthProvider } from 'hooks/useAuth'
import { TimerProvider } from 'hooks/useTimer'

import 'styles/globals.css'

// Poppins, served with the app (next/font downloads it at build time, so no
// request goes to Google from the browser). Japanese, which Poppins doesn't
// have, falls back to the system's Japanese font (globals.css).
const poppins = Poppins({
  subsets: ['latin'],
  weight: ['400', '500', '600', '700'],
  display: 'swap',
})

export default function App({ Component, pageProps }: AppProps) {
  return (
    <>
      {/* On :root rather than a wrapper, so pop-ups and the page share it */}
      <style jsx global>{`
        :root {
          --font-sans: ${poppins.style.fontFamily};
        }
      `}</style>
      <AuthProvider>
        <TimerProvider>
          <Component {...pageProps} />
        </TimerProvider>
      </AuthProvider>
    </>
  )
}
