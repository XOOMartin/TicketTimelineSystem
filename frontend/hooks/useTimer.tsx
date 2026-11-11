import { ReactNode, createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'

import ConfirmDialog from 'components/molecules/ConfirmDialog'
import Toast from 'components/molecules/Toast'
import useAuth from 'hooks/useAuth'
import useOptions from 'hooks/useOptions'
import * as api from 'lib/api'
import { formatClock, formatShortHours } from 'lib/dates'
import { sameTicket, ticketKeyOf } from 'lib/tickets'
import { errorMessage } from 'lib/errors'
import { StoppedTimer, Timer, TimerInput } from 'types/worklog'

type TimerState = {
  timer: Timer | null
  // Milliseconds worked, ticking every second while it counts; pauses left out
  elapsed: number
  isPaused: boolean
  isBusy: boolean
  // Both resolve to the timer that was ended (start ends a running one
  // first, after asking), or null; they reject if the server says no. Only
  // one timer runs at a time, so the same minutes are never counted twice.
  start: (input: TimerInput) => Promise<StoppedTimer | null>
  stop: () => Promise<StoppedTimer | null>
  // Asks "End the timer?" first, so a misclick can't end it; resolves to
  // null if the user keeps timing
  confirmStop: () => Promise<StoppedTimer | null>
  pause: () => Promise<void>
  resume: () => Promise<void>
  // For buttons: pause or resume, whichever applies, and end (asking
  // first). A refusal shows in the timer's toast, so there's nothing to catch.
  togglePause: () => void
  end: () => void
  // Goes up each time ending a timer logs time, so pages can reload
  loggedCount: number
}

const TimerContext = createContext<TimerState | null>(null)

// Set in the tab's session storage as a page goes away with a timer running.
// Only the same tab sees it again, so a page that finds it was reloaded (or
// went back to the site); one that doesn't was opened fresh.
const LEAVING_KEY = 'ticket-timeline-timer-leaving'

// Whether this page load follows a page of the same tab going away; clears it
function cameBackInSameTab(): boolean {
  try {
    const left = sessionStorage.getItem(LEAVING_KEY) !== null
    sessionStorage.removeItem(LEAVING_KEY)
    return left
  } catch {
    return false
  }
}

// The running timer is shared by the header and the check-in form. It
// belongs to the open site: it ends when End is clicked, or when the site is
// closed (its time recorded up to the close). The server keeps its start, so
// it carries on across pages, reloads and other open tabs.
export function TimerProvider(props: { children: ReactNode }) {
  const { user } = useAuth()
  const { phaseLabel } = useOptions()
  const [timer, setTimer] = useState<Timer | null>(null)
  const [now, setNow] = useState(() => Date.now())
  const [isBusy, setIsBusy] = useState(false)
  const [loggedCount, setLoggedCount] = useState(0)
  const [notice, setNotice] = useState<{ text: string; tone: 'success' | 'error' } | null>(null)
  const clearNotice = useCallback(() => setNotice(null), [])

  // Says the site is open: the timer carries on, or it ended while the site
  // was closed, which is reported once
  const reportClosed = useRef<(s: StoppedTimer) => void>(() => {})
  const sayOpen = useCallback(
    (fresh = false) =>
      api
        .openTimer(fresh)
        .then((res) => {
          setTimer(res.timer)
          if (res.stopped) reportClosed.current(res.stopped)
        })
        .catch(() => setTimer(null)),
    [],
  )

  const cameBack = useRef<boolean | null>(null)
  useEffect(() => {
    if (!user) {
      setTimer(null)
      return
    }
    // A reload carries the timer on. A page opened fresh after the site was
    // closed ends it at the close, so closing the tab really stops it. Read
    // once per page load (development runs this effect twice).
    cameBack.current ??= cameBackInSameTab()
    if (cameBack.current) {
      sayOpen()
      // Again in a moment: the page that went away may have told the server
      // it was closing after this one said it's open
      const again = setTimeout(sayOpen, 3000)
      return () => clearTimeout(again)
    }
    sayOpen(true)
  }, [user, sayOpen])

  // Closing the site ends the timer. A page going away can't tell a close
  // from a reload, so it tells the server it's closing, and the server waits
  // a little for a page to open again. Other open tabs hear about it and say
  // they're still open, so closing one of several doesn't end it.
  const timerRef = useRef(timer)
  timerRef.current = timer
  useEffect(() => {
    if (!user) return
    const tabs = typeof BroadcastChannel === 'undefined' ? null : new BroadcastChannel('ticket-timeline-timer')
    const onHide = () => {
      if (!timerRef.current) return
      try {
        sessionStorage.setItem(LEAVING_KEY, String(Date.now()))
      } catch {
        // No storage: a reload then counts as closing, which ends the timer
      }
      api.sendTimerClosing()
      tabs?.postMessage('closing')
    }
    // Twice: the closing tab's beacon may reach the server after the first
    const onOtherClosing = () => {
      sayOpen()
      setTimeout(() => sayOpen(), 3000)
    }
    // Back from the browser's back/forward cache, the page didn't load again
    const onShow = (e: PageTransitionEvent) => {
      if (e.persisted) {
        cameBackInSameTab()
        sayOpen()
      }
    }
    // While a timer runs, closing the tab or browser asks first ("Leave
    // site?"), since leaving ends it. Browsers word the question themselves.
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      if (!timerRef.current) return
      e.preventDefault()
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    window.addEventListener('pagehide', onHide)
    window.addEventListener('pageshow', onShow)
    tabs?.addEventListener('message', onOtherClosing)
    return () => {
      window.removeEventListener('beforeunload', onBeforeUnload)
      window.removeEventListener('pagehide', onHide)
      window.removeEventListener('pageshow', onShow)
      tabs?.close()
    }
  }, [user, sayOpen])

  // Ticks only while counting; a paused clock stands still
  useEffect(() => {
    setNow(Date.now())
    if (!timer || timer.pausedAt) return
    const tick = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(tick)
  }, [timer])

  const describe = useCallback((t: Timer) => `${phaseLabel(t.phase)} · ${t.ticketKey}`, [phaseLabel])

  // Runs a timer action, reporting failures in a toast and re-throwing
  const run = useCallback(async <T,>(action: () => Promise<T>): Promise<T> => {
    setIsBusy(true)
    try {
      return await action()
    } catch (e) {
      setNotice({ text: errorMessage(e), tone: 'error' })
      throw e
    } finally {
      setIsBusy(false)
    }
  }, [])

  const logged = useCallback(
    (s: StoppedTimer) => {
      if (s.minutes > 0) setLoggedCount((n) => n + 1)
      return s.minutes > 0
        ? `Logged ${formatShortHours(s.minutes / 60)} on ${describe(s.timer)}`
        : `Under a minute on ${describe(s.timer)}, so nothing was logged`
    },
    [describe],
  )

  reportClosed.current = (s: StoppedTimer) => {
    const at = new Date(s.timer.closingAt ?? Date.now()).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    setNotice({
      text: `The site was closed at ${at}, so the timer ended. ${logged(s)}`,
      tone: s.minutes > 0 ? 'success' : 'error',
    })
  }

  const startNow = useCallback(
    (input: TimerInput) =>
      run(async () => {
        const res = await api.startTimer(input)
        setTimer(res.timer)
        const started = res.timer ? `Timing ${describe(res.timer)}` : ''
        setNotice({ text: res.stopped ? `${logged(res.stopped)}. ${started}` : started, tone: 'success' })
        return res.stopped ?? null
      }),
    [run, describe, logged],
  )

  // Starting while another timer runs ends that one, so it asks first (with
  // a warning when it's under a minute, which records nothing). Keeping the
  // running timer resolves to null and starts nothing.
  const [switching, setSwitching] = useState<TimerInput | null>(null)
  const settleSwitch = useRef<((stopped: StoppedTimer | null) => void) & { reject?: (e: unknown) => void }>(() => {})
  const start = useCallback(
    (input: TimerInput) => {
      const same =
        timer !== null && sameTicket(timer, input) && timer.round === input.round && timer.phase === input.phase
      if (!timer || same) return startNow(input)
      return new Promise<StoppedTimer | null>((resolve, reject) => {
        // A refusal from the server still reaches the caller
        settleSwitch.current = Object.assign(resolve, { reject })
        setSwitching(input)
      })
    },
    [timer, startNow],
  )

  const stop = useCallback(
    () =>
      run(async () => {
        const res = await api.stopTimer()
        setTimer(null)
        setNotice({ text: logged(res), tone: 'success' })
        return res
      }),
    [run, logged],
  )

  const pause = useCallback(
    () =>
      run(async () => {
        const res = await api.pauseTimer()
        setTimer(res.timer)
        if (res.timer) setNotice({ text: `Paused ${describe(res.timer)}`, tone: 'success' })
      }),
    [run, describe],
  )

  const resume = useCallback(
    () =>
      run(async () => {
        const res = await api.resumeTimer()
        setTimer(res.timer)
        if (res.timer) setNotice({ text: `Timing ${describe(res.timer)} again`, tone: 'success' })
      }),
    [run, describe],
  )

  // The question before ending; settles the promise confirmStop returned
  const [isConfirming, setIsConfirming] = useState(false)
  const settle = useRef<(stopped: StoppedTimer | null) => void>(() => {})
  const confirmStop = useCallback(
    () =>
      new Promise<StoppedTimer | null>((resolve) => {
        settle.current = resolve
        setIsConfirming(true)
      }),
    [],
  )

  const togglePause = useCallback(() => {
    ;(timer?.pausedAt ? resume() : pause()).catch(() => {})
  }, [timer?.pausedAt, pause, resume])
  const end = useCallback(() => {
    confirmStop().catch(() => {})
  }, [confirmStop])

  const elapsed = timer
    ? (timer.pausedAt ? new Date(timer.pausedAt).getTime() : now) -
      new Date(timer.startedAt).getTime() -
      timer.pausedSeconds * 1000
    : 0
  const isPaused = Boolean(timer?.pausedAt)
  const value = useMemo(
    () => ({
      timer,
      elapsed,
      isPaused,
      isBusy,
      start,
      stop,
      confirmStop,
      pause,
      resume,
      togglePause,
      end,
      loggedCount,
    }),
    [timer, elapsed, isPaused, isBusy, start, stop, confirmStop, pause, resume, togglePause, end, loggedCount],
  )

  return (
    <TimerContext.Provider value={value}>
      {props.children}
      {switching && timer && (
        <ConfirmDialog
          title="Switch the timer?"
          confirmLabel="End it and start"
          busyLabel="Switching…"
          cancelLabel="Keep timing"
          tone="primary"
          onConfirm={async () => {
            try {
              settleSwitch.current(await startNow(switching))
            } catch (e) {
              settleSwitch.current.reject?.(e)
              throw e
            }
            settleSwitch.current = () => {}
          }}
          onClose={() => {
            setSwitching(null)
            // Closed without switching: nothing started, the caller gets null
            settleSwitch.current(null)
            settleSwitch.current = () => {}
          }}
        >
          Only one timer runs at a time. <strong>{describe(timer)}</strong> has been timing for{' '}
          <strong>{formatClock(Math.max(0, elapsed))}</strong>
          {elapsed < 60_000 ? (
            <>
              , which is <strong>under a minute, so it won’t be recorded</strong>
            </>
          ) : (
            ': ending it records that time'
          )}
          , then{' '}
          <strong>
            {phaseLabel(switching.phase)} · {ticketKeyOf(switching)}
          </strong>{' '}
          starts.
        </ConfirmDialog>
      )}
      {isConfirming && timer && (
        <ConfirmDialog
          title="End the timer?"
          confirmLabel="End timer"
          busyLabel="Ending…"
          cancelLabel="Keep timing"
          tone="primary"
          onConfirm={async () => {
            const stopped = await stop()
            settle.current(stopped)
            settle.current = () => {}
          }}
          onClose={() => {
            setIsConfirming(false)
            // Closed without ending: the caller gets null
            settle.current(null)
            settle.current = () => {}
          }}
        >
          {elapsed < 60_000 ? (
            <>
              <strong>{formatClock(Math.max(0, elapsed))}</strong> on <strong>{describe(timer)}</strong> is under a
              minute, so <strong>nothing will be recorded</strong>.
            </>
          ) : (
            <>
              <strong>{formatClock(Math.max(0, elapsed))}</strong> on <strong>{describe(timer)}</strong> will be
              recorded{isPaused ? ' (it is paused; the pause is left out)' : ''}.
            </>
          )}
        </ConfirmDialog>
      )}
      <Toast message={notice?.text ?? null} tone={notice?.tone} onDone={clearNotice} />
    </TimerContext.Provider>
  )
}

export default function useTimer(): TimerState {
  const ctx = useContext(TimerContext)
  if (!ctx) throw new Error('useTimer must be used inside TimerProvider')
  return ctx
}
