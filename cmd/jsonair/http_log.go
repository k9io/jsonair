/**
 ** Copyright (C) 2026 Key9, Inc <k9.io>
 ** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
 **
 ** This file is part of the JSONAir.
 **
 ** This source code is licensed under the MIT license found in the
 ** LICENSE file in the root directory of this source tree.
 **
 **/

package main

import (
	"time"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
)

func HTTP_Logger() gin.HandlerFunc {

	return func(c *gin.Context) {

		clientIP := c.ClientIP()

		now := time.Now()

		l.Logger(l.INFO, "[%s] %s %s %s", now.Format(time.RFC3339), c.Request.Method, c.Request.URL.Path, clientIP)
		c.Next()

	}
}
