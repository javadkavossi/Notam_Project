const API_BASE = import.meta.env.VITE_API_BASE ?? ''
const BASE = `${API_BASE}/api/v1`
const AUTH_STORAGE_KEY = 'notam-auth'

function getAuthHeaders(): HeadersInit {
  try {
    const raw = localStorage.getItem(AUTH_STORAGE_KEY)
    if (!raw) return {}
    const u = JSON.parse(raw) as { token?: string }
    if (u?.token) return { Authorization: `Bearer ${u.token}` }
  } catch {
    // ignore
  }
  return {}
}

export interface NotamItem {
  id: number
  messageId: string
  seriesNumber: string
  eventType: string
  locationIcao: string
  airportIcao?: string
  airportName?: string
  affectedFir?: string
  effectiveStart: string
  effectiveEnd?: string | null
  plainText: string
  formattedText?: string
  lowerLimit?: string
  upperLimit?: string
  createdAt: string
}

export interface NotamFilters {
  seriesNumber?: string
  eventType?: string
  locationIcao?: string
  airportIcao?: string
  airportName?: string
  affectedFir?: string
  plainText?: string
  from?: string
  to?: string
  limit?: number
  offset?: number
}

export interface ListResponse {
  items: NotamItem[]
  totalCount: number
}

export interface AlertOptionsResponse {
  firs: string[]
  airports: string[]
}

export interface AlertSettings {
  selectedFirs: string[]
  selectedAirports: string[]
  selectedKeywords: string[]
  customKeywords: string[]
}

/** کلیدواژه‌های قابل انتخاب برای فیلتر اعلان (در متن NOTAM جستجو می‌شوند) */
export const ALERT_KEYWORDS = [
  'AD CLSD',
  'RWY',
  'ILS',
  'GPS',
  'SID',
  'STAR',
  'VOR',
  'DME',
  'SECTOR',
] as const

function defaultAlertSettings(): AlertSettings {
  return { selectedFirs: [], selectedAirports: [], selectedKeywords: [], customKeywords: [] }
}

const ALERT_SETTINGS_KEY = 'notam-alert-settings'

/** خواندن تنظیمات اعلان از localStorage (برای فرم سایدبار) */
export function getStoredAlertSettings(): AlertSettings {
  try {
    const raw = localStorage.getItem(ALERT_SETTINGS_KEY)
    if (!raw) return defaultAlertSettings()
    const parsed = JSON.parse(raw) as Partial<AlertSettings>
    return {
      selectedFirs: Array.isArray(parsed.selectedFirs) ? parsed.selectedFirs : [],
      selectedAirports: Array.isArray(parsed.selectedAirports) ? parsed.selectedAirports : [],
      selectedKeywords: Array.isArray(parsed.selectedKeywords) ? parsed.selectedKeywords : [],
      customKeywords: Array.isArray(parsed.customKeywords) ? parsed.customKeywords : [],
    }
  } catch {
    return defaultAlertSettings()
  }
}

/** ذخیره تنظیمات اعلان در localStorage (برای فرم سایدبار) */
export function setStoredAlertSettings(s: AlertSettings): void {
  localStorage.setItem(ALERT_SETTINGS_KEY, JSON.stringify(s))
}

/** دریافت تنظیمات اعلان از سرور (نیاز به لاگین) */
export async function fetchAlertSettings(): Promise<AlertSettings> {
  const res = await fetch(`${BASE}/notams/alert-settings`, { headers: getAuthHeaders() })
  const data = await res.json()
  if (!res.ok) throw new Error(data.error || 'خطا در دریافت تنظیمات')
  if (!data.success || data.result == null) return defaultAlertSettings()
  const r = data.result as { selectedFirs?: string[]; selectedAirports?: string[]; selectedKeywords?: string[]; customKeywords?: string[] }
  return {
    selectedFirs: Array.isArray(r.selectedFirs) ? r.selectedFirs : [],
    selectedAirports: Array.isArray(r.selectedAirports) ? r.selectedAirports : [],
    selectedKeywords: Array.isArray(r.selectedKeywords) ? r.selectedKeywords : [],
    customKeywords: Array.isArray(r.customKeywords) ? r.customKeywords : [],
  }
}

/** ذخیره تنظیمات اعلان در سرور (نیاز به لاگین) */
export async function saveAlertSettings(s: AlertSettings): Promise<void> {
  const res = await fetch(`${BASE}/notams/alert-settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
    body: JSON.stringify({
      selectedFirs: s.selectedFirs,
      selectedAirports: s.selectedAirports,
      selectedKeywords: s.selectedKeywords,
      customKeywords: s.customKeywords ?? [],
    }),
  })
  const data = await res.json()
  if (!res.ok || !data.success) throw new Error(data.error || 'خطا در ذخیره تنظیمات')
}

export async function fetchAlertOptions(): Promise<AlertOptionsResponse> {
  const res = await fetch(`${BASE}/notams/alert-options`)
  const data = await res.json()
  if (!data.success || !data.result) {
    throw new Error(data.error || 'خطا در دریافت گزینه‌ها')
  }
  return data.result
}

/** NOTAMهای جدیدی که با تنظیمات اعلان کاربر در دیتابیس مطابقت دارند (نیاز به لاگین) */
export async function fetchRecentNotams(sinceSeconds = 120, limit = 20): Promise<ListResponse> {
  const params = new URLSearchParams()
  params.set('since_seconds', String(sinceSeconds))
  params.set('limit', String(limit))
  const res = await fetch(`${BASE}/notams/recent?${params}`, { headers: getAuthHeaders() })
  const data = await res.json()
  if (!data.success || !data.result) {
    throw new Error(data.error || 'خطا در دریافت NOTAMهای جدید')
  }
  return data.result
}

export async function fetchNotams(filters: NotamFilters = {}): Promise<ListResponse> {
  const params = new URLSearchParams()
  Object.entries(filters).forEach(([k, v]) => {
    if (v != null && v !== '') params.set(k, String(v))
  })
  const url = `${BASE}/notams?${params}`
  const res = await fetch(url)
  const data = await res.json()
  if (!data.success || !data.result) {
    throw new Error(data.error || 'خطا در دریافت داده‌ها')
  }
  return data.result
}

/** دریافت یک NOTAM با شماره سریال (برای نمایش NOTAM لغو‌شده در پاپ‌آپ) */
export async function fetchNotamBySeriesNumber(series: string): Promise<NotamItem | null> {
  const res = await fetch(`${BASE}/notams/by-series?series=${encodeURIComponent(series)}`)
  const data = await res.json()
  if (res.status === 404 || !data.success) return null
  if (!data.result) return null
  return data.result as NotamItem
}
