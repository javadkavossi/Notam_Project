import { useState, useEffect } from 'react'
import {
  fetchAlertOptions,
  fetchAlertSettings,
  saveAlertSettings,
  ALERT_KEYWORDS,
  type AlertSettings,
} from '../api/client'
import './AlertSettingsModal.css'

interface Props {
  open: boolean
  onClose: () => void
  onSave?: (s: AlertSettings) => void
}

export default function AlertSettingsModal({ open, onClose, onSave }: Props) {
  const [firs, setFirs] = useState<string[]>([])
  const [airports, setAirports] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [saved, setSaved] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [settings, setSettings] = useState<AlertSettings>({ selectedFirs: [], selectedAirports: [], selectedKeywords: [], customKeywords: [] })
  const [customKeywordInput, setCustomKeywordInput] = useState('')

  useEffect(() => {
    if (!open) return
    setSaveError('')
    let cancelled = false
    Promise.all([fetchAlertOptions(), fetchAlertSettings()])
      .then(([options, currentSettings]) => {
        if (!cancelled) {
          setFirs([...(options.firs ?? [])].sort((a, b) => a.localeCompare(b)))
          setAirports([...(options.airports ?? [])].sort((a, b) => a.localeCompare(b)))
          setSettings(currentSettings)
        }
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [open])

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

  const toggleKeyword = (kw: string) => {
    const next = settings.selectedKeywords.includes(kw)
      ? settings.selectedKeywords.filter((k) => k !== kw)
      : [...settings.selectedKeywords, kw]
    setSettings((s) => ({ ...s, selectedKeywords: next }))
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
  const selectAllKeywords = () => {
    setSettings((s) => ({ ...s, selectedKeywords: [...ALERT_KEYWORDS] }))
    setSaved(false)
  }
  const deselectAllKeywords = () => {
    setSettings((s) => ({ ...s, selectedKeywords: [] }))
    setSaved(false)
  }
  const allFirsSelected = firs.length > 0 && settings.selectedFirs.length === firs.length
  const allAirportsSelected = airports.length > 0 && settings.selectedAirports.length === airports.length
  const allKeywordsSelected = ALERT_KEYWORDS.length > 0 && settings.selectedKeywords.length === ALERT_KEYWORDS.length

  const addCustomKeyword = () => {
    const kw = customKeywordInput.trim()
    if (!kw || settings.customKeywords.includes(kw)) return
    setSettings((s) => ({ ...s, customKeywords: [...s.customKeywords, kw] }))
    setCustomKeywordInput('')
    setSaved(false)
  }
  const removeCustomKeyword = (kw: string) => {
    setSettings((s) => ({ ...s, customKeywords: s.customKeywords.filter((k) => k !== kw) }))
    setSaved(false)
  }

  const save = async () => {
    setSaveError('')
    try {
      await saveAlertSettings(settings)
      setSaved(true)
      onSave?.(settings)
      setTimeout(() => {
        setSaved(false)
        onClose()
      }, 800)
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'خطا در ذخیره')
    }
  }

  if (!open) return null

  return (
    <div className="alert-modal-overlay" onClick={onClose} role="dialog" aria-modal="true" aria-labelledby="alert-modal-title">
      <div className="alert-modal" onClick={(e) => e.stopPropagation()}>
        <div className="alert-modal-header">
          <h2 id="alert-modal-title">تنظیمات اعلان NOTAM</h2>
          <button type="button" className="alert-modal-close" onClick={onClose} aria-label="بستن">
            ×
          </button>
        </div>
        <div className="alert-modal-body">
          <p className="alert-modal-desc">
            اعلان زمانی داده می‌شود که NOTAM هم مربوط به یکی از FIR/فرودگاه‌های انتخاب‌شده باشد و هم در متن آن حداقل یکی از کلیدواژه‌های انتخاب‌شده وجود داشته باشد
            (مثال: FIR ایران + فرودگاه مهرآباد OIII + کلیدواژه STAR).
          </p>
          {loading ? (
            <p className="alert-modal-loading">در حال بارگذاری گزینه‌ها...</p>
          ) : (
            <>
              <div className="alert-modal-section">
                <div className="alert-modal-section-header">
                  <label>FIR (مرتب‌شده)</label>
                  <button type="button" className="btn-select-all" onClick={allFirsSelected ? deselectAllFirs : selectAllFirs}>
                    {allFirsSelected ? 'پاک کردن همه' : 'انتخاب همه'}
                  </button>
                </div>
                <div className="alert-modal-chips">
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
              <div className="alert-modal-section">
                <div className="alert-modal-section-header">
                  <label>فرودگاه ICAO (مرتب‌شده)</label>
                  <button type="button" className="btn-select-all" onClick={allAirportsSelected ? deselectAllAirports : selectAllAirports}>
                    {allAirportsSelected ? 'پاک کردن همه' : 'انتخاب همه'}
                  </button>
                </div>
                <div className="alert-modal-chips">
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
              <div className="alert-modal-section">
                <div className="alert-modal-section-header">
                  <label>کلیدواژه در متن NOTAM (حداقل یکی در پیام باید باشد)</label>
                  <button type="button" className="btn-select-all" onClick={allKeywordsSelected ? deselectAllKeywords : selectAllKeywords}>
                    {allKeywordsSelected ? 'پاک کردن همه' : 'انتخاب همه'}
                  </button>
                </div>
                <div className="alert-modal-keywords">
                  {ALERT_KEYWORDS.map((kw) => (
                    <label key={kw} className="alert-keyword-check">
                      <input
                        type="checkbox"
                        checked={settings.selectedKeywords.includes(kw)}
                        onChange={() => toggleKeyword(kw)}
                      />
                      <span>{kw}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div className="alert-modal-section">
                <label>کلیدواژهٔ سفارشی (افزودن توسط کاربر)</label>
                <div className="alert-modal-custom-keywords">
                  <input
                    type="text"
                    className="alert-modal-keyword-input"
                    placeholder="مثال: STAR یا ILS"
                    value={customKeywordInput}
                    onChange={(e) => setCustomKeywordInput(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addCustomKeyword())}
                  />
                  <button type="button" className="btn-add-keyword" onClick={addCustomKeyword}>
                    افزودن
                  </button>
                </div>
                {settings.customKeywords.length > 0 && (
                  <div className="alert-modal-custom-list">
                    {settings.customKeywords.map((kw) => (
                      <span key={kw} className="custom-keyword-tag">
                        {kw}
                        <button type="button" className="custom-keyword-remove" onClick={() => removeCustomKeyword(kw)} aria-label="حذف">
                          ×
                        </button>
                      </span>
                    ))}
                  </div>
                )}
              </div>
            </>
          )}
        </div>
        <div className="alert-modal-footer">
          {saveError && <p className="alert-modal-error">{saveError}</p>}
          <button type="button" className="btn-alert-cancel" onClick={onClose}>
            انصراف
          </button>
          <button type="button" className="btn-alert-save" onClick={save} disabled={loading}>
            {saved ? '✓ ذخیره شد' : 'ذخیره تنظیمات'}
          </button>
        </div>
      </div>
    </div>
  )
}
