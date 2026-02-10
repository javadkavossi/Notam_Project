import { useState, useEffect, useCallback, useRef } from 'react'
import { toast } from 'react-toastify'
import { useAuth } from '../contexts/AuthContext'
import {
  fetchNotams,
  fetchRecentNotams,
  type NotamItem,
  type NotamFilters,
  type AlertSettings,
} from '../api/client'
import NotamFiltersForm from '../components/NotamFiltersForm'
import NotamList from '../components/NotamList'
import PaginationBar from '../components/PaginationBar'
import AlertSettingsModal from '../components/AlertSettingsModal'
import NotamNotificationContent from '../components/NotamNotificationContent'
import { playAlertSound, unlockAudio } from '../utils/alertSound'
import './Dashboard.css'

const AUTO_REFRESH_MS = 8000
const ALERT_POLL_MS = 2000
const ALERT_RECENT_SECONDS = 120
const SOUND_UNLOCK_KEY = 'notam-sound-unlocked'

function getSoundUnlocked(): boolean {
  try {
    return localStorage.getItem(SOUND_UNLOCK_KEY) === '1'
  } catch {
    return false
  }
}

function setSoundUnlocked(): void {
  try {
    localStorage.setItem(SOUND_UNLOCK_KEY, '1')
  } catch {}
}

export default function Dashboard() {
  const { user, logout } = useAuth()
  const [notams, setNotams] = useState<NotamItem[]>([])
  const [totalCount, setTotalCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [filters, setFilters] = useState<NotamFilters>({ limit: 5, offset: 0 })
  const [alertSettings, setAlertSettings] = useState<AlertSettings>({ selectedFirs: [], selectedAirports: [], selectedKeywords: [] })
  const [soundUnlocked, setSoundUnlockedState] = useState(getSoundUnlocked())
  const [alertModalOpen, setAlertModalOpen] = useState(false)
  const shownAlertIds = useRef<Set<string>>(new Set())
  const MAX_SHOWN_IDS = 200

  const loadNotams = useCallback(async () => {
    try {
      setError('')
      const res = await fetchNotams(filters)
      setNotams(res.items)
      setTotalCount(res.totalCount)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'خطا در بارگذاری')
    } finally {
      setLoading(false)
    }
  }, [filters])

  useEffect(() => {
    setLoading(true)
    loadNotams()
  }, [loadNotams])

  useEffect(() => {
    const id = setInterval(loadNotams, AUTO_REFRESH_MS)
    return () => clearInterval(id)
  }, [loadNotams])

  const showNotamToast = useCallback((item: NotamItem) => {
    const type = item.eventType === 'C' ? 'error' : item.eventType === 'R' ? 'warning' : 'success'
    toast(
      ({ closeToast, toastProps }) => (
        <NotamNotificationContent
          notam={(toastProps?.data as NotamItem) ?? item}
          closeToast={closeToast}
        />
      ),
      {
        type,
        position: 'top-center',
        autoClose: false,
        closeButton: false,
        data: item,
      }
    )
    if (getSoundUnlocked()) playAlertSound()
  }, [])

  const testNotification = useCallback(() => {
    const fake: NotamItem = {
      id: 0,
      messageId: 'test-' + Date.now(),
      seriesNumber: 'A1234/24',
      eventType: 'N',
      locationIcao: 'OIII',
      airportIcao: 'OIII',
      airportName: 'تهران امام',
      affectedFir: 'OIIX',
      effectiveStart: new Date().toISOString(),
      effectiveEnd: new Date(Date.now() + 86400000).toISOString(),
      plainText: 'TEST NOTAM - این یک نوتیف تست است.',
      createdAt: new Date().toISOString(),
    }
    showNotamToast(fake)
  }, [showNotamToast])

  useEffect(() => {
    const id = setInterval(async () => {
      try {
        const res = await fetchRecentNotams(ALERT_RECENT_SECONDS, 20)
        for (const item of res.items) {
          if (!item.messageId || shownAlertIds.current.has(item.messageId)) continue
          shownAlertIds.current.add(item.messageId)
          if (shownAlertIds.current.size > MAX_SHOWN_IDS) {
            const arr = Array.from(shownAlertIds.current)
            shownAlertIds.current = new Set(arr.slice(-MAX_SHOWN_IDS / 2))
          }
          showNotamToast(item)
        }
      } catch {
        // ignore
      }
    }, ALERT_POLL_MS)
    return () => clearInterval(id)
  }, [showNotamToast])

  const onFilterChange = (newFilters: NotamFilters) => {
    setFilters((prev) => ({ ...newFilters, limit: prev.limit, offset: 0 }))
  }

  const onPageChange = useCallback((page: number) => {
    setFilters((prev) => ({ ...prev, offset: (page - 1) * (prev.limit ?? 5) }))
  }, [])

  const onPageSizeChange = useCallback((limit: number) => {
    setFilters((prev) => ({ ...prev, limit, offset: 0 }))
  }, [])

  const onUnlockSound = () => {
    unlockAudio()
    setSoundUnlocked()
    setSoundUnlockedState(true)
  }

  return (
    <div className="dashboard">
      {!soundUnlocked && (
        <div className="sound-banner">
          <span>برای پخش صدای اعلان NOTAM، اجازه پخش صدا را فعال کنید.</span>
          <button type="button" className="btn-unlock-sound" onClick={onUnlockSound}>
            فعال‌سازی صدا
          </button>
        </div>
      )}
      <header className="dashboard-header">
        <div>
          <h1>NOTAM Viewer</h1>
          <span className="user">خوش آمدید، {user?.user}</span>
        </div>
        <div className="dashboard-header-actions">
          <button type="button" className="btn-alert-settings" onClick={testNotification}>
            تست نوتیفیکیشن
          </button>
          <button type="button" className="btn-alert-settings" onClick={() => setAlertModalOpen(true)}>
            تنظیمات اعلان
          </button>
          <button onClick={logout} className="btn-logout">
            خروج
          </button>
        </div>
      </header>

      <div className="dashboard-content">
        <aside className="filters-panel">
          <h3>فیلترها</h3>
          <NotamFiltersForm filters={filters} onChange={onFilterChange} onApply={loadNotams} />
        </aside>

        <main className="main-panel">
          <div className="stats">
            <span>تعداد کل: {totalCount}</span>
            <span className="auto-refresh">به‌روزرسانی خودکار هر {AUTO_REFRESH_MS / 1000} ثانیه</span>
          </div>
          {error && <p className="error">{error}</p>}
          <PaginationBar
            totalCount={totalCount}
            limit={filters.limit ?? 5}
            offset={filters.offset ?? 0}
            onPageChange={onPageChange}
            onPageSizeChange={onPageSizeChange}
            loading={loading}
          />
          {loading && notams.length === 0 ? (
            <p className="loading">در حال بارگذاری...</p>
          ) : (
            <NotamList notams={notams} loading={loading} />
          )}
          <PaginationBar
            totalCount={totalCount}
            limit={filters.limit ?? 5}
            offset={filters.offset ?? 0}
            onPageChange={onPageChange}
            onPageSizeChange={onPageSizeChange}
            loading={loading}
          />
        </main>
      </div>

      <AlertSettingsModal
        open={alertModalOpen}
        onClose={() => setAlertModalOpen(false)}
        onSave={(s) => setAlertSettings(s)}
      />
    </div>
  )
}
