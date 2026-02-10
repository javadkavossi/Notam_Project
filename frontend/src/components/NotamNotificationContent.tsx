import { useState, useCallback } from 'react'
import type { NotamItem } from '../api/client'
import './NotamNotification.css'

interface Props {
  notam: NotamItem
  closeToast?: () => void
}

function formatDate(s: string): string {
  try {
    const d = new Date(s)
    return d.toLocaleString('en-GB', { dateStyle: 'short', timeStyle: 'short' })
  } catch {
    return s
  }
}

function Copyable({ value, label }: { value: string; label: string }) {
  const [copied, setCopied] = useState(false)
  const copy = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation()
      if (!value) return
      navigator.clipboard.writeText(value).then(
        () => {
          setCopied(true)
          setTimeout(() => setCopied(false), 2000)
        },
        () => {}
      )
    },
    [value]
  )
  if (!value) return null
  return (
    <button type="button" className="notam-notif-copyable" onClick={copy} title="Click to copy">
      <span className="notam-notif-label">{label}:</span>
      <span className="notam-notif-value">{value}</span>
      {copied ? <span className="notam-notif-copied">✓ Copied</span> : <span className="notam-notif-hint">Click to copy</span>}
    </button>
  )
}

const EVENT_TITLE: Record<string, string> = { N: 'New NOTAM', R: 'Replacement NOTAM', C: 'Cancelled NOTAM' }

export default function NotamNotificationContent({ notam, closeToast }: Props) {
  const n = notam
  const title = (n.eventType && EVENT_TITLE[n.eventType]) || 'NOTAM'

  return (
    <div className="notam-notif">
      <div className="notam-notif-header">
        <span className="notam-notif-title">{title}</span>
        <button type="button" className="notam-notif-close" onClick={closeToast} aria-label="Close">
          ×
        </button>
      </div>
      <div className="notam-notif-body">
        <div className="notam-notif-copy-row">
          <Copyable value={n.seriesNumber} label="Serial" />
          <Copyable value={n.affectedFir ?? ''} label="FIR" />
        </div>
        <div className="notam-notif-meta">
          {n.airportIcao && <span className="notam-notif-badge">{n.airportIcao}</span>}
          {n.locationIcao && n.locationIcao !== n.airportIcao && (
            <span className="notam-notif-badge">{n.locationIcao}</span>
          )}
        </div>
        {n.airportName && (
          <div className="notam-notif-row">
            <span className="notam-notif-label">Airport:</span> {n.airportName}
          </div>
        )}
        <div className="notam-notif-row">
          <span className="notam-notif-label">Validity:</span>{' '}
          {formatDate(n.effectiveStart)}
          {n.effectiveEnd && ` to ${formatDate(n.effectiveEnd)}`}
        </div>
        {(n.lowerLimit != null || n.upperLimit != null) && (
          <div className="notam-notif-row">
            <span className="notam-notif-label">Range:</span>{' '}
            {[n.lowerLimit, n.upperLimit].filter(Boolean).join(' - ')}
          </div>
        )}
        <pre className="notam-notif-text">{n.plainText}</pre>
      </div>
    </div>
  )
}
