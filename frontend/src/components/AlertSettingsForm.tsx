import { useState, useEffect } from 'react'
import {
  fetchAlertOptions,
  getStoredAlertSettings,
  setStoredAlertSettings,
  type AlertSettings,
} from '../api/client'
import './AlertSettingsForm.css'

interface Props {
  onSettingsChange?: (s: AlertSettings) => void
}

export default function AlertSettingsForm({ onSettingsChange }: Props) {
  const [firs, setFirs] = useState<string[]>([])
  const [airports, setAirports] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [saved, setSaved] = useState(false)
  const [settings, setSettings] = useState<AlertSettings>(getStoredAlertSettings())

  useEffect(() => {
    let cancelled = false
    fetchAlertOptions()
      .then((res) => {
        if (!cancelled) {
          setFirs(res.firs ?? [])
          setAirports(res.airports ?? [])
        }
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [])

  const toggleFir = (code: string) => {
    const next = settings.selectedFirs.includes(code)
      ? settings.selectedFirs.filter((c) => c !== code)
      : [...settings.selectedFirs, code]
    setSettings((s) => ({ ...s, selectedFirs: next }))
    setSaved(false)
  }

  const toggleAirport = (code: string) => {
    const next = settings.selectedAirports.includes(code)
      ? settings.selectedAirports.filter((c) => c !== code)
      : [...settings.selectedAirports, code]
    setSettings((s) => ({ ...s, selectedAirports: next }))
    setSaved(false)
  }

  const selectAllFirs = () => {
    setSettings((s) => ({ ...s, selectedFirs: [...firs] }))
    setSaved(false)
  }
  const deselectAllFirs = () => {
    setSettings((s) => ({ ...s, selectedFirs: [] }))
    setSaved(false)
  }
  const selectAllAirports = () => {
    setSettings((s) => ({ ...s, selectedAirports: [...airports] }))
    setSaved(false)
  }
  const deselectAllAirports = () => {
    setSettings((s) => ({ ...s, selectedAirports: [] }))
    setSaved(false)
  }
  const allFirsSelected = firs.length > 0 && settings.selectedFirs.length === firs.length
  const allAirportsSelected = airports.length > 0 && settings.selectedAirports.length === airports.length

  const save = () => {
    setStoredAlertSettings(settings)
    setSaved(true)
    onSettingsChange?.(settings)
    setTimeout(() => setSaved(false), 2000)
  }

  if (loading) {
    return (
      <div className="alert-settings-form">
        <h4>تنظیمات اعلان</h4>
        <p className="alert-settings-loading">در حال بارگذاری گزینه‌ها...</p>
      </div>
    )
  }

  return (
    <div className="alert-settings-form">
      <h4>تنظیمات اعلان NOTAM</h4>
      <p className="alert-settings-desc">
        فقط برای FIRها و فرودگاه‌های انتخاب‌شده زیر، با دریافت NOTAM جدید پاپ‌آپ و صدا پخش می‌شود.
      </p>
      <div className="alert-settings-row">
        <div className="alert-settings-row-header">
          <label>FIR</label>
          <span className="alert-settings-actions">
            {allFirsSelected ? (
              <button type="button" className="btn-select-all" onClick={deselectAllFirs}>
                پاک کردن همه
              </button>
            ) : (
              <button type="button" className="btn-select-all" onClick={selectAllFirs}>
                انتخاب همه
              </button>
            )}
          </span>
        </div>
        <div className="alert-settings-chips">
          {firs.map((code) => (
            <button
              key={code}
              type="button"
              className={`chip ${settings.selectedFirs.includes(code) ? 'chip-active' : ''}`}
              onClick={() => toggleFir(code)}
            >
              {code}
            </button>
          ))}
        </div>
      </div>
      <div className="alert-settings-row">
        <div className="alert-settings-row-header">
          <label>فرودگاه (ICAO)</label>
          <span className="alert-settings-actions">
            {allAirportsSelected ? (
              <button type="button" className="btn-select-all" onClick={deselectAllAirports}>
                پاک کردن همه
              </button>
            ) : (
              <button type="button" className="btn-select-all" onClick={selectAllAirports}>
                انتخاب همه
              </button>
            )}
          </span>
        </div>
        <div className="alert-settings-chips">
          {airports.map((code) => (
            <button
              key={code}
              type="button"
              className={`chip ${settings.selectedAirports.includes(code) ? 'chip-active' : ''}`}
              onClick={() => toggleAirport(code)}
            >
              {code}
            </button>
          ))}
        </div>
      </div>
      <button type="button" className="btn-alert-save" onClick={save}>
        {saved ? '✓ ذخیره شد' : 'ذخیره تنظیمات'}
      </button>
    </div>
  )
}
