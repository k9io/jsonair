package main

import (
	"github.com/gin-gonic/gin"
	"time"
)

func HTTP_Logger() gin.HandlerFunc {

	return func(c *gin.Context) {

		clientIP := c.ClientIP()

		now := time.Now()

		Logger(INFO, "[%s] %s %s %s", now.Format(time.RFC3339), c.Request.Method, c.Request.URL.Path, clientIP)
		c.Next()

	}
}
