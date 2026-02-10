import './PaginationBar.css'

export const PAGE_SIZES = [5, 10, 20, 30, 40, 50] as const

interface Props {
  totalCount: number
  limit: number
  offset: number
  onPageChange: (page: number) => void
  onPageSizeChange: (limit: number) => void
  loading?: boolean
}

export default function PaginationBar({
  totalCount,
  limit,
  offset,
  onPageChange,
  onPageSizeChange,
  loading = false,
}: Props) {
  const totalPages = Math.max(1, Math.ceil(totalCount / limit))
  const currentPage = totalPages === 0 ? 1 : Math.min(totalPages, Math.floor(offset / limit) + 1)
  const from = totalCount === 0 ? 0 : offset + 1
  const to = Math.min(offset + limit, totalCount)

  const goTo = (page: number) => {
    const p = Math.max(1, Math.min(totalPages, page))
    onPageChange(p)
  }

  return (
    <div className="pagination-bar" role="navigation" aria-label="صفحه‌بندی">
      <div className="pagination-per-page">
        <label htmlFor="pagination-size">تعداد در هر صفحه:</label>
        <select
          id="pagination-size"
          value={limit}
          onChange={(e) => onPageSizeChange(Number(e.target.value))}
          disabled={loading}
          aria-label="تعداد در هر صفحه"
        >
          {PAGE_SIZES.map((size) => (
            <option key={size} value={size}>
              {size}
            </option>
          ))}
        </select>
      </div>
      <div className="pagination-range">
        {totalCount === 0 ? (
          <span>۰ مورد</span>
        ) : (
          <span>
            {from.toLocaleString('fa-IR')} – {to.toLocaleString('fa-IR')} از {totalCount.toLocaleString('fa-IR')}
          </span>
        )}
      </div>
      <div className="pagination-nav">
        <button
          type="button"
          className="pagination-btn"
          onClick={() => goTo(1)}
          disabled={currentPage <= 1 || loading}
          aria-label="صفحه اول"
        >
          اولین
        </button>
        <button
          type="button"
          className="pagination-btn"
          onClick={() => goTo(currentPage - 1)}
          disabled={currentPage <= 1 || loading}
          aria-label="صفحه قبل"
        >
          قبلی
        </button>
        <span className="pagination-page-info">
          صفحه {currentPage.toLocaleString('fa-IR')} از {totalPages.toLocaleString('fa-IR')}
        </span>
        <button
          type="button"
          className="pagination-btn"
          onClick={() => goTo(currentPage + 1)}
          disabled={currentPage >= totalPages || loading}
          aria-label="صفحه بعد"
        >
          بعدی
        </button>
        <button
          type="button"
          className="pagination-btn"
          onClick={() => goTo(totalPages)}
          disabled={currentPage >= totalPages || loading}
          aria-label="صفحه آخر"
        >
          آخرین
        </button>
      </div>
    </div>
  )
}
