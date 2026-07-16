package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/helper"
	"github.com/hossein-repo/BaseProject/api/middleware"
	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/briefing"
)

// FlightHandler تعریف پرواز و دریافت بریفینگ (E5).
type FlightHandler struct {
	svc *briefing.Service
}

func NewFlightHandler() *FlightHandler {
	return &FlightHandler{svc: briefing.NewService()}
}

// FlightRequest بدنهٔ ساخت پرواز.
type FlightRequest struct {
	ADEP          string   `json:"adep" binding:"required"`
	ADES          string   `json:"ades" binding:"required"`
	Alternates    []string `json:"alternates"`
	EnrouteFirs   []string `json:"enrouteFirs"`
	ETD           string   `json:"etd" binding:"required"` // RFC3339
	ETA           string   `json:"eta" binding:"required"` // RFC3339
	BufferMinutes int      `json:"bufferMinutes"`
	Note          string   `json:"note"`
}

// toModel بدنه را به مدل تبدیل و اعتبارسنجی می‌کند.
func (r FlightRequest) toModel(username string) (model.FlightPlan, error) {
	etd, err := time.Parse(time.RFC3339, r.ETD)
	if err != nil {
		return model.FlightPlan{}, err
	}
	eta, err := time.Parse(time.RFC3339, r.ETA)
	if err != nil {
		return model.FlightPlan{}, err
	}
	buf := r.BufferMinutes
	if buf <= 0 {
		buf = 60
	}
	if buf > 24*60 {
		buf = 24 * 60
	}
	return model.FlightPlan{
		Username:      username,
		ADEP:          strings.ToUpper(strings.TrimSpace(r.ADEP)),
		ADES:          strings.ToUpper(strings.TrimSpace(r.ADES)),
		Alternates:    model.StringSlice(upperAll(r.Alternates)),
		EnrouteFIRs:   model.StringSlice(upperAll(r.EnrouteFirs)),
		ETD:           etd.UTC(),
		ETA:           eta.UTC(),
		BufferMinutes: buf,
		Note:          r.Note,
	}, nil
}

func upperAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if v := strings.ToUpper(strings.TrimSpace(s)); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// CreateFlight godoc
// @Summary ساخت پرواز جدید
// @Description تعریف مبدأ/مقصد/الترنت‌ها/FIRهای مسیر و پنجرهٔ زمانی
// @Tags flights
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body FlightRequest true "پرواز"
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/flights [post]
func (h *FlightHandler) CreateFlight(c *gin.Context) {
	username := middleware.GetUsername(c)
	var req FlightRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, helper.GenerateBaseResponseWithError(nil, false, helper.ValidationError, err))
		return
	}
	fp, err := req.toModel(username)
	if err != nil {
		c.JSON(http.StatusBadRequest, helper.GenerateBaseResponseWithError(nil, false, helper.ValidationError, err))
		return
	}
	if fp.ETA.Before(fp.ETD) {
		c.JSON(http.StatusBadRequest, helper.GenerateBaseResponse(nil, false, helper.ValidationError))
		return
	}
	if err := h.svc.CreateFlight(&fp); err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(fp, true, helper.Success))
}

// ListFlights godoc
// @Summary لیست پروازهای کاربر
// @Tags flights
// @Security BearerAuth
// @Produce json
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/flights [get]
func (h *FlightHandler) ListFlights(c *gin.Context) {
	items, err := h.svc.ListFlights(middleware.GetUsername(c), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(items, true, helper.Success))
}

// Briefing godoc
// @Summary بریفینگ یک پرواز ذخیره‌شده
// @Description NOTAMهای مرتبط با پرواز، امتیازدهی و گروه‌بندی‌شده
// @Tags flights
// @Security BearerAuth
// @Produce json
// @Param id path int true "شناسه پرواز"
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/flights/{id}/briefing [get]
func (h *FlightHandler) Briefing(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, helper.GenerateBaseResponseWithError(nil, false, helper.ValidationError, err))
		return
	}
	fp, err := h.svc.FindFlight(id, middleware.GetUsername(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	if fp == nil {
		c.JSON(http.StatusNotFound, helper.GenerateBaseResponse(nil, false, helper.NotFoundError))
		return
	}
	b, err := h.svc.Build(*fp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(b, true, helper.Success))
}

// PreviewBriefing godoc
// @Summary بریفینگ فوری بدون ذخیرهٔ پرواز
// @Description برای گرفتن بریفینگ سریع از روی یک تعریف پرواز موقت
// @Tags flights
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body FlightRequest true "پرواز"
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/flights/briefing [post]
func (h *FlightHandler) PreviewBriefing(c *gin.Context) {
	var req FlightRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, helper.GenerateBaseResponseWithError(nil, false, helper.ValidationError, err))
		return
	}
	fp, err := req.toModel(middleware.GetUsername(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, helper.GenerateBaseResponseWithError(nil, false, helper.ValidationError, err))
		return
	}
	b, err := h.svc.Build(fp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(b, true, helper.Success))
}
