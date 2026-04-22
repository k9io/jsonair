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
	"net/http"
	"sync"
	"time"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	limiters   = make(map[string]*ipLimiter)
	limitersMu sync.Mutex
)

func getLimiter(ip string) *rate.Limiter {
	limitersMu.Lock()
	defer limitersMu.Unlock()

	entry, exists := limiters[ip]
	if !exists {
		entry = &ipLimiter{
			limiter: rate.NewLimiter(rate.Every(time.Minute/5), 5), // 5 attempts per minute
		}
		limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func cleanLimiters() {
	limitersMu.Lock()
	defer limitersMu.Unlock()
	for ip, entry := range limiters {
		if time.Since(entry.lastSeen) > 10*time.Minute {
			delete(limiters, ip)
		}
	}
}

func rateLimitMiddleware() gin.HandlerFunc {
	// Periodically clean up stale entries to prevent unbounded memory growth.
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			cleanLimiters()
		}
	}()

	return func(c *gin.Context) {
		limiter := getLimiter(c.ClientIP())
		if !limiter.Allow() {
			l.Logger(l.WARN, "Rate limit exceeded for %s on auth endpoint.", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}
		c.Next()
	}
}
