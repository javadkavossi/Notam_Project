import { useState, useCallback } from 'react'
import type { NotamItem } from '../api/client'
import './NotamAlertPopup.css'

interface Props {
  notam: NotamItem
  onClose: () => void
}

function formatDate(s: string): string {
  try {
    const d = new Date(s)
    return d.toLocaleString('fa-IR', { dateStyle: 'short', timeStyle: 'short' })
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
    <button
      type="button"
      className="notam-alert-copyable"
      onClick={copy}
      title="کلیک برای کپی"
      aria-label={`کپی ${label}: ${value}`}
    >
      <span className="notam-alert-copy-label">{label}:</span>
      <span className="notam-alert-copy-value">{value}</span>
      {copied ? (
        <span className="notam-alert-copied">✓ کپی شد</span>
      ) : (
        <span className="notam-alert-copy-hint">کلیک برای کپی</span>
      )}
    </button>
  )
}

export default function NotamAlertPopup({ notam, onClose }: Props) {
  const n = notam

  return (
    <div className="notam-alert-overlay" role="dialog" aria-modal="true" aria-labelledby="notam-alert-title">
      <div className="notam-alert-popup">
        <div className="notam-alert-header">
          <h2 id="notam-alert-title">جزئیات NOTAM</h2>
          <button type="button" className="notam-alert-close" onClick={onClose} aria-label="بستن">
            ×
          </button>
        </div>
        <div className="notam-alert-body">
          <div className="notam-alert-meta">
            <Copyable value={n.seriesNumber} label="سریال" />
            <Copyable value={n.affectedFir ?? ''} label="FIR" />
            {n.airportIcao && <span className="notam-alert-badge">{n.airportIcao}</span>}
            {n.locationIcao && n.locationIcao !== n.airportIcao && (
              <span className="notam-alert-badge">{n.locationIcao}</span>
            )}
            {n.eventType && (
              <span className="notam-alert-type" data-type={n.eventType}>
                {n.eventType === 'N' && 'جدید'}
                {n.eventType === 'R' && 'جایگزین'}
                {n.eventType === 'C' && 'لغو'}
                {!['N', 'R', 'C'].includes(n.eventType) && n.eventType}
              </span>
            )}
          </div>
          {n.airportName && (
            <div className="notam-alert-row">
              <span className="notam-alert-row-label">فرودگاه:</span> {n.airportName}
            </div>
          )}
          <div className="notam-alert-row">
            <span className="notam-alert-row-label">اعتبار:</span>{' '}
            {formatDate(n.effectiveStart)}
            {n.effectiveEnd && ` تا ${formatDate(n.effectiveEnd)}`}
          </div>
          {(n.lowerLimit != null || n.upperLimit != null) && (
            <div className="notam-alert-row">
              <span className="notam-alert-row-label">محدوده:</span>{' '}
              {[n.lowerLimit, n.upperLimit].filter(Boolean).join(' - ')}
            </div>
          )}
          <pre className="notam-alert-text">{n.plainText}</pre>
        </div>
        <div className="notam-alert-footer">
          <button type="button" className="btn-alert-close" onClick={onClose}>
            بستن
          </button>
        </div>
      </div>
    </div>
  )
}
