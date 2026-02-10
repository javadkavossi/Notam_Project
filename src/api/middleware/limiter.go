package middleware

import (
	"net/http"

	"github.com/didip/tollbooth"
	"github.com/gin-gonic/gin"
)

func LimitByRequest() gin.HandlerFunc {
	lmt := tollbooth.NewLimiter(1, nil)

	return func(c *gin.Context) {
		// بررسی محدودیت
		err := tollbooth.LimitByRequest(lmt, c.Writer, c.Request)
		if err != nil {
			// اگه بیشتر از حد مجاز بود:
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success":    false,
				"resultCode": "LIMITER_ERROR",
				"message":    "Too many requests, please try again later.",
				"error":      err.Error(),
			})
			return
		}

		// ادامه‌ی مسیر
		c.Next()
	}
}
