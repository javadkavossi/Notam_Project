import { useState } from 'react'
import type { NotamItem } from '../api/client'
import './NotamList.css'

interface Props {
  notams: NotamItem[]
  loading?: boolean
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

export default function NotamList({ notams, loading }: Props) {
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
            {n.affectedFir && <span className="fir">FIR: {n.affectedFir}</span>}
          </div>
          <div className="notam-meta">
            <span>از: {formatDate(n.effectiveStart)}</span>
            {n.effectiveEnd && <span>تا: {formatDate(n.effectiveEnd)}</span>}
            {n.airportName && <span>{n.airportName}</span>}
          </div>
          <p className="notam-preview">{n.plainText.slice(0, 150)}{n.plainText.length > 150 ? '...' : ''}</p>
          {expandedId === n.id && (
            <div className="notam-detail">
              <pre className="formatted-text">{n.formattedText || n.plainText}</pre>
            </div>
          )}
          <span className="expand-hint">{expandedId === n.id ? 'کلیک برای بستن' : 'کلیک برای جزئیات'}</span>
        </div>
      ))}
    </div>
  )
}
