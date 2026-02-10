package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/helper"
	"github.com/hossein-repo/BaseProject/api/middleware"
	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/messaging"
	"gorm.io/gorm"
)

// NotamHandler API برای بازخوانی NOTAMها (فرمت ICAO/Jeppesen)
type NotamHandler struct{}

// NewNotamHandler returns a new NotamHandler
func NewNotamHandler() *NotamHandler {
	return &NotamHandler{}
}

// ListResponse پاسخ لیست NOTAMها
type ListResponse struct {
	Items      []NotamItem `json:"items"`
	TotalCount int64       `json:"totalCount"`
}

// NotamItem یک NOTAM برای خروجی API (سازگار با استاندارد ICAO)
type NotamItem struct {
	ID             uint       `json:"id"`
	MessageID      string     `json:"messageId"`
	SeriesNumber   string     `json:"seriesNumber"`
	EventType      string     `json:"eventType"`
	LocationICAO   string     `json:"locationIcao"`
	AirportICAO    string     `json:"airportIcao,omitempty"`
	AirportName    string     `json:"airportName,omitempty"`
	AffectedFIR    string     `json:"affectedFir,omitempty"`
	EffectiveStart time.Time  `json:"effectiveStart"`
	EffectiveEnd   *time.Time `json:"effectiveEnd,omitempty"`
	PlainText      string     `json:"plainText"`
	FormattedText  string     `json:"formattedText,omitempty"`
	LowerLimit     string     `json:"lowerLimit,omitempty"`
	UpperLimit     string     `json:"upperLimit,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// AlertOptionsResponse گزینه‌های تنظیم اعلان (FIR و فرودگاه)
type AlertOptionsResponse struct {
	Firs     []string `json:"firs"`
	Airports []string `json:"airports"`
}

// AlertOptions godoc
// @Summary Get alert filter options
// @Description لیست FIRها و فرودگاه‌های مجاز برای تنظیم اعلان
// @Tags notams
// @Produce json
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/notams/alert-options [get]
func (h *NotamHandler) AlertOptions(c *gin.Context) {
	firs := make([]string, len(messaging.AllowedFIRs))
	copy(firs, messaging.AllowedFIRs)
	airports := make([]string, len(messaging.AllowedAirports))
	copy(airports, messaging.AllowedAirports)
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(AlertOptionsResponse{
		Firs:     firs,
		Airports: airports,
	}, true, helper.Success))
}

// AlertSettingsBody بدنهٔ درخواست ذخیره تنظیمات اعلان
type AlertSettingsBody struct {
	SelectedFirs     []string `json:"selectedFirs"`
	SelectedAirports []string `json:"selectedAirports"`
	SelectedKeywords []string `json:"selectedKeywords"`
	CustomKeywords   []string `json:"customKeywords"`
}

// GetAlertSettings godoc
// @Summary Get current user alert settings
// @Description دریافت تنظیمات اعلان کاربر جاری از دیتابیس
// @Tags notams
// @Security BearerAuth
// @Produce json
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/notams/alert-settings [get]
func (h *NotamHandler) GetAlertSettings(c *gin.Context) {
	username := middleware.GetUsername(c)
	if username == "" {
		c.JSON(http.StatusUnauthorized, helper.GenerateBaseResponse(nil, false, helper.AuthError))
		return
	}
	var s model.NotamAlertSettings
	err := db.GetDb().Where("username = ?", username).First(&s).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, helper.GenerateBaseResponse(AlertSettingsBody{
			SelectedFirs:     nil,
			SelectedAirports: nil,
			SelectedKeywords: nil,
			CustomKeywords:   nil,
		}, true, helper.Success))
		return
	}
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(AlertSettingsBody{
		SelectedFirs:     s.SelectedFirs,
		SelectedAirports: s.SelectedAirports,
		SelectedKeywords: s.SelectedKeywords,
		CustomKeywords:   s.CustomKeywords,
	}, true, helper.Success))
}

// SaveAlertSettings godoc
// @Summary Save current user alert settings
// @Description ذخیره تنظیمات اعلان کاربر جاری در دیتابیس
// @Tags notams
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body AlertSettingsBody true "تنظیمات"
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/notams/alert-settings [put]
func (h *NotamHandler) SaveAlertSettings(c *gin.Context) {
	username := middleware.GetUsername(c)
	if username == "" {
		c.JSON(http.StatusUnauthorized, helper.GenerateBaseResponse(nil, false, helper.AuthError))
		return
	}
	var body AlertSettingsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, helper.GenerateBaseResponseWithError(nil, false, helper.ValidationError, err))
		return
	}
	db := db.GetDb()
	var s model.NotamAlertSettings
	err := db.Where("username = ?", username).First(&s).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	s.Username = username
	s.SelectedFirs = body.SelectedFirs
	s.SelectedAirports = body.SelectedAirports
	s.SelectedKeywords = body.SelectedKeywords
	if body.CustomKeywords != nil {
		s.CustomKeywords = body.CustomKeywords
	} else {
		s.CustomKeywords = nil
	}
	if s.Id == 0 {
		if err := db.Create(&s).Error; err != nil {
			c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
			return
		}
	} else {
		if err := db.Save(&s).Error; err != nil {
			c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
			return
		}
	}
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(AlertSettingsBody{
		SelectedFirs:     s.SelectedFirs,
		SelectedAirports: s.SelectedAirports,
		SelectedKeywords: s.SelectedKeywords,
		CustomKeywords:   s.CustomKeywords,
	}, true, helper.Success))
}

// Recent godoc
// @Summary List NOTAMs that matched user alert settings (delivered when message arrived)
// @Description NOTAMهایی که هنگام رسیدن از consumer با تنظیمات اعلان کاربر مطابقت داشتند و ثبت تحویل شده‌اند
// @Tags notams
// @Security BearerAuth
// @Produce json
// @Param since_seconds query int false "تحویل‌های N ثانیه اخیر (پیش‌فرض 120)"
// @Param limit query int false "حداکثر تعداد (پیش‌فرض 50)"
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/notams/recent [get]
func (h *NotamHandler) Recent(c *gin.Context) {
	username := middleware.GetUsername(c)
	if username == "" {
		c.JSON(http.StatusUnauthorized, helper.GenerateBaseResponse(nil, false, helper.AuthError))
		return
	}
	database := db.GetDb()
	sinceSec := 120
	if v := c.Query("since_seconds"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 600 {
			sinceSec = n
		}
	}
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	since := time.Now().Add(-time.Duration(sinceSec) * time.Second)
	var deliveries []model.NotamAlertDelivery
	err := database.Where("username = ? AND created_at >= ?", username, since).
		Order("created_at DESC").
		Limit(limit).
		Find(&deliveries).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	if len(deliveries) == 0 {
		c.JSON(http.StatusOK, helper.GenerateBaseResponse(ListResponse{Items: nil, TotalCount: 0}, true, helper.Success))
		return
	}
	notamIds := make([]int, 0, len(deliveries))
	for _, d := range deliveries {
		notamIds = append(notamIds, d.NotamId)
	}
	var notams []model.Notam
	if err := database.Where("id IN ?", notamIds).Find(&notams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	byId := make(map[int]model.Notam)
	for _, n := range notams {
		byId[n.Id] = n
	}
	items := make([]NotamItem, 0, len(deliveries))
	for _, d := range deliveries {
		if n, ok := byId[d.NotamId]; ok {
			items = append(items, notamToItem(n))
		}
	}
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(ListResponse{
		Items:      items,
		TotalCount: int64(len(items)),
	}, true, helper.Success))
}

// List godoc
// @Summary List NOTAMs
// @Description لیست NOTAMهای ذخیره‌شده با فیلتر اختیاری (فرمت ICAO)
// @Tags notams
// @Accept json
// @Produce json
// @Param location_icao query string false "فیلتر کد ICAO فرودگاه (مثلاً OIII)"
// @Param from query string false "از تاریخ (YYYY-MM-DD)"
// @Param to query string false "تا تاریخ (YYYY-MM-DD)"
// @Param limit query int false "حداکثر تعداد (پیش‌فرض 50)"
// @Param offset query int false "جابجایی برای صفحه‌بندی"
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/notams [get]
func (h *NotamHandler) List(c *gin.Context) {
	database := db.GetDb()

	query := database.Model(&model.Notam{})

	// helper: camelCase یا snake_case
	q := func(camel, snake string) string {
		if v := c.Query(camel); v != "" {
			return v
		}
		return c.Query(snake)
	}

	// فیلترها بر اساس ساختار دیتابیس (camelCase و snake_case)
	if v := q("seriesNumber", "series_number"); v != "" {
		query = query.Where("series_number ILIKE ?", "%"+v+"%")
	}
	if v := q("eventType", "event_type"); v != "" {
		query = query.Where("event_type = ?", v)
	}
	if v := q("locationIcao", "location_icao"); v != "" {
		query = query.Where("location_icao ILIKE ?", "%"+v+"%")
	}
	if v := q("airportIcao", "airport_icao"); v != "" {
		query = query.Where("airport_icao ILIKE ?", "%"+v+"%")
	}
	if v := q("airportName", "airport_name"); v != "" {
		query = query.Where("airport_name ILIKE ?", "%"+v+"%")
	}
	if v := q("affectedFir", "affected_fir"); v != "" {
		query = query.Where("affected_fir ILIKE ?", "%"+v+"%")
	}
	if v := q("plainText", "plain_text"); v != "" {
		query = query.Where("plain_text ILIKE ?", "%"+v+"%")
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			query = query.Where("effective_start >= ?", t)
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			end := t.Add(24 * time.Hour)
			query = query.Where("effective_start < ?", end)
		}
	}

	// شمارش کل
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}

	// صفحه‌بندی
	limit := 50
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	// مرتب‌سازی: جدیدترین اول
	query = query.Order("effective_start DESC").Limit(limit).Offset(offset)

	var notams []model.Notam
	if err := query.Find(&notams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}

	items := make([]NotamItem, 0, len(notams))
	for _, n := range notams {
		items = append(items, notamToItem(n))
	}

	c.JSON(http.StatusOK, helper.GenerateBaseResponse(ListResponse{
		Items:      items,
		TotalCount: total,
	}, true, helper.Success))
}

// GetByID godoc
// @Summary Get NOTAM by ID
// @Description دریافت یک NOTAM با شناسه
// @Tags notams
// @Accept json
// @Produce json
// @Param id path int true "شناسه NOTAM"
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/notams/{id} [get]
func (h *NotamHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, helper.GenerateBaseResponseWithError(nil, false, helper.ValidationError, err))
		return
	}

	var n model.Notam
	if err := db.GetDb().First(&n, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, helper.GenerateBaseResponse(nil, false, helper.NotFoundError))
			return
		}
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}

	c.JSON(http.StatusOK, helper.GenerateBaseResponse(notamToItem(n), true, helper.Success))
}

func notamToItem(n model.Notam) NotamItem {
	return NotamItem{
		ID:             uint(n.Id),
		MessageID:      n.MessageID,
		SeriesNumber:   n.SeriesNumber,
		EventType:      n.EventType,
		LocationICAO:   n.LocationICAO,
		AirportICAO:    n.AirportICAO,
		AirportName:    n.AirportName,
		AffectedFIR:    n.AffectedFIR,
		EffectiveStart: n.EffectiveStart,
		EffectiveEnd:   n.EffectiveEnd,
		PlainText:      n.PlainText,
		FormattedText:  n.FormattedText,
		LowerLimit:     n.LowerLimit,
		UpperLimit:     n.UpperLimit,
		CreatedAt:      n.CreatedAt,
	}
}
