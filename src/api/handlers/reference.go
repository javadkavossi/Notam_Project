package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/helper"
	"github.com/hossein-repo/BaseProject/internal/reference"
)

// ReferenceHandler endpointهای دادهٔ مرجع (فرودگاه/باند/…) — E7-5.
type ReferenceHandler struct {
	store *reference.Store
}

func NewReferenceHandler() *ReferenceHandler {
	return &ReferenceHandler{store: reference.NewStore()}
}

// AirportSearch godoc
// @Summary جستجوی فرودگاه برای autocomplete
// @Description تطبیق روی ICAO/IATA/نام؛ برای فرم تعریف پرواز
// @Tags reference
// @Produce json
// @Param q query string true "متن جستجو (مثلاً OII)"
// @Param limit query int false "حداکثر نتایج (پیش‌فرض 10)"
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/reference/airports [get]
func (h *ReferenceHandler) AirportSearch(c *gin.Context) {
	q := c.Query("q")
	limit := 10
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	items, err := h.store.SearchAirports(q, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(items, true, helper.Success))
}

// AirportByICAO godoc
// @Summary دریافت یک فرودگاه با کد ICAO
// @Tags reference
// @Produce json
// @Param icao path string true "کد ICAO"
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/reference/airports/{icao} [get]
func (h *ReferenceHandler) AirportByICAO(c *gin.Context) {
	a, err := h.store.FindAirport(c.Param("icao"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	if a == nil {
		c.JSON(http.StatusNotFound, helper.GenerateBaseResponse(nil, false, helper.NotFoundError))
		return
	}
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(a, true, helper.Success))
}
