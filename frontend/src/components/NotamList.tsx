import { useState } from 'react'
import type { NotamItem } from '../api/client'
import {
  getCancelledNotamRef,
  getCancelledNotamDisplayText,
  getReplacedNotamRef,
  getReplacedNotamDisplayText,
  stripQLineFromFormattedText,
} from '../utils/notamCancel'
import './NotamList.css'

interface Props {
  notams: NotamItem[]
  loading?: boolean
  onOpenCancelledNotam?: (series: string) => void
  onOpenReplacedNotam?: (series: string) => void
  loadingCancelled?: boolean
}

const eventTypeLabel: Record<string, string> = {
  N: 'جدید',
  R: 'جایگزین',
  C: 'لغو',
}

function formatDate(s: string) {
  try {
    return new Date(s).toLocaleString('fa-IR', {
      dateStyle: 'short',
      timeStyle: 'short',
    })
  } catch {
    return s
  }
}

export default function NotamList({
  notams,
  loading,
  onOpenCancelledNotam,
  onOpenReplacedNotam,
  loadingCancelled,
}: Props) {
  const [expandedId, setExpandedId] = useState<number | null>(null)

  if (notams.length === 0 && !loading) {
    return (
      <div className="notam-empty">
        <p>NOTAMی یافت نشد</p>
      </div>
    )
  }

  return (
    <div className={`notam-list ${loading ? 'loading' : ''}`}>
      {notams.map((n) => (
        <div
          key={n.id}
          className={`notam-card ${expandedId === n.id ? 'expanded' : ''}`}
          onClick={() => setExpandedId(expandedId === n.id ? null : n.id)}
        >
          <div className="notam-header">
            <span className="series">{n.seriesNumber || '-'}</span>
            <span className={`event-type event-${n.eventType}`}>
              {eventTypeLabel[n.eventType] || n.eventType}
            </span>
            <span className="location">{n.locationIcao}</span>
            {n.airportIcao && n.airportIcao !== n.locationIcao && (
              <span className="airport-icao">({n.airportIcao})</span>
            )}
            {n.eventType !== 'C' && n.affectedFir && <span className="fir">FIR: {n.affectedFir}</span>}
          </div>
          {n.eventType === 'C' && getCancelledNotamRef(n.formattedText, n.plainText) && (
            <div className="notam-cancelled-ref">
              NOTAM لغو‌شده:{' '}
              {onOpenCancelledNotam ? (
                <button
                  type="button"
                  className="notam-cancelled-ref-btn"
                  onClick={(e) => {
                    e.stopPropagation()
                    onOpenCancelledNotam(getCancelledNotamRef(n.formattedText, n.plainText)!)
                  }}
                  disabled={loadingCancelled}
                  title="مشاهده جزئیات NOTAM لغو‌شده"
                >
                  {getCancelledNotamRef(n.formattedText, n.plainText)}
                </button>
              ) : (
                getCancelledNotamRef(n.formattedText, n.plainText)
              )}
            </div>
          )}
          {n.eventType === 'R' && getReplacedNotamRef(n.formattedText, n.plainText) && (
            <div className="notam-replaced-ref">
              NOTAM جایگزین‌شده:{' '}
              {onOpenReplacedNotam ? (
                <button
                  type="button"
                  className="notam-replaced-ref-btn"
                  onClick={(e) => {
                    e.stopPropagation()
                    onOpenReplacedNotam(getReplacedNotamRef(n.formattedText, n.plainText)!)
                  }}
                  disabled={loadingCancelled}
                  title="مشاهده جزئیات NOTAM جایگزین‌شده"
                >
                  {getReplacedNotamRef(n.formattedText, n.plainText)}
                </button>
              ) : (
                getReplacedNotamRef(n.formattedText, n.plainText)
              )}
            </div>
          )}
          <div className="notam-meta">
            <span>از: {formatDate(n.effectiveStart)}</span>
            {n.effectiveEnd && <span>تا: {formatDate(n.effectiveEnd)}</span>}
            {n.airportName && <span>{n.airportName}</span>}
          </div>
          <p className="notam-preview">
            {n.eventType === 'C'
              ? getCancelledNotamDisplayText(n.plainText, n.formattedText).slice(0, 150)
              : n.eventType === 'R'
                ? getReplacedNotamDisplayText(n.plainText, n.formattedText).slice(0, 150)
                : n.plainText.slice(0, 150)}
            {(n.eventType === 'C'
              ? getCancelledNotamDisplayText(n.plainText, n.formattedText).length
              : n.eventType === 'R'
                ? getReplacedNotamDisplayText(n.plainText, n.formattedText).length
                : n.plainText.length) > 150
              ? '...'
              : ''}
          </p>
          {expandedId === n.id && (
            <div className="notam-detail">
              <pre className="formatted-text">
                {n.eventType === 'C'
                  ? stripQLineFromFormattedText(n.formattedText || n.plainText) || getCancelledNotamDisplayText(n.plainText, n.formattedText)
                  : n.eventType === 'R'
                    ? n.formattedText || getReplacedNotamDisplayText(n.plainText, n.formattedText)
                    : n.formattedText || n.plainText}
              </pre>
            </div>
          )}
          <span className="expand-hint">{expandedId === n.id ? 'کلیک برای بستن' : 'کلیک برای جزئیات'}</span>
        </div>
      ))}
    </div>
  )
}
