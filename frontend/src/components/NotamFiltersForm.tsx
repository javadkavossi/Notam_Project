import { useState, useEffect } from 'react'
import type { NotamFilters } from '../api/client'
import './NotamFiltersForm.css'

interface Props {
  filters: NotamFilters
  onChange: (f: NotamFilters) => void
  onApply: () => void
}

export default function NotamFiltersForm({ filters, onChange, onApply }: Props) {
  const [local, setLocal] = useState<NotamFilters>(filters)

  useEffect(() => {
    setLocal(filters)
  }, [filters])

  const update = (key: keyof NotamFilters, value: string | number | undefined) => {
    const next = { ...local, [key]: value || undefined }
    setLocal(next)
    onChange(next)
  }

  const clear = () => {
    const empty: NotamFilters = { limit: 5, offset: 0 }
    setLocal(empty)
    onChange(empty)
  }

  const eventTypeLabel: Record<string, string> = {
    N: 'جدید',
    R: 'جایگزین',
    C: 'لغو',
  }

  return (
    <div className="filters-form">
      <div className="filter-row">
        <label>جستجوی FIR</label>
        <input
          type="text"
          placeholder="مثال: OIIX"
          value={local.affectedFir ?? ''}
          onChange={(e) => update('affectedFir', e.target.value)}
        />
      </div>
      <div className="filter-row">
        <label>کد فرودگاه / Location</label>
        <input
          type="text"
          placeholder="مثال: OIII, CDB"
          value={local.locationIcao ?? ''}
          onChange={(e) => update('locationIcao', e.target.value)}
        />
      </div>
      <div className="filter-row">
        <label>کد ICAO فرودگاه</label>
        <input
          type="text"
          placeholder="مثال: OIIE"
          value={local.airportIcao ?? ''}
          onChange={(e) => update('airportIcao', e.target.value)}
        />
      </div>
      <div className="filter-row">
        <label>نام فرودگاه</label>
        <input
          type="text"
          placeholder="نام فرودگاه"
          value={local.airportName ?? ''}
          onChange={(e) => update('airportName', e.target.value)}
        />
      </div>
      <div className="filter-row">
        <label>شماره سریال</label>
        <input
          type="text"
          placeholder="مثال: A1477/26"
          value={local.seriesNumber ?? ''}
          onChange={(e) => update('seriesNumber', e.target.value)}
        />
      </div>
      <div className="filter-row">
        <label>نوع رویداد</label>
        <select
          value={local.eventType ?? ''}
          onChange={(e) => update('eventType', e.target.value || undefined)}
        >
          <option value="">همه</option>
          <option value="N">{eventTypeLabel.N}</option>
          <option value="R">{eventTypeLabel.R}</option>
          <option value="C">{eventTypeLabel.C}</option>
        </select>
      </div>
      <div className="filter-row">
        <label>از تاریخ</label>
        <input
          type="date"
          value={local.from ?? ''}
          onChange={(e) => update('from', e.target.value || undefined)}
        />
      </div>
      <div className="filter-row">
        <label>تا تاریخ</label>
        <input
          type="date"
          value={local.to ?? ''}
          onChange={(e) => update('to', e.target.value || undefined)}
        />
      </div>
      <div className="filter-row">
        <label>جستجو در متن</label>
        <input
          type="text"
          placeholder="متن NOTAM"
          value={local.plainText ?? ''}
          onChange={(e) => update('plainText', e.target.value)}
        />
      </div>
      <div className="filter-actions">
        <button onClick={onApply} className="btn-apply">
          اعمال
        </button>
        <button onClick={clear} className="btn-clear">
          پاک کردن
        </button>
      </div>
    </div>
  )
}
