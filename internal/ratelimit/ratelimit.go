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

/* Package ratelimit provides the per-IP rate limiter used on the PAT -> JWT
   exchange endpoint.  It is shared by jsonair and jsonair-write and is
   configured with the RATE_LIMIT_* environment variables. */

package ratelimit

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	DefaultPerMinute = 5
	DefaultBurst     = 5

	idleTimeout     = 10 * time.Minute
	cleanupInterval = 5 * time.Minute
)

type Config struct {
	Enabled   bool
	PerMinute int            /* Sustained requests per minute, per client IP */
	Burst     int            /* Requests allowed at once before the per-minute rate applies */
	Exempt    []netip.Prefix /* Client IPs / networks that are never rate limited */
}

/* ParseEnv reads RATE_LIMIT_ENABLED, RATE_LIMIT_PER_MINUTE, RATE_LIMIT_BURST and
   RATE_LIMIT_EXEMPT using getenv (normally os.Getenv).  Unset values fall back
   to the defaults, which match JSONAir's historical behavior. */

func ParseEnv(getenv func(string) string) (Config, error) {

	cfg := Config{
		Enabled:   true,
		PerMinute: DefaultPerMinute,
		Burst:     DefaultBurst,
	}

	switch v := strings.ToLower(strings.TrimSpace(getenv("RATE_LIMIT_ENABLED"))); v {
	case "", "true":
		cfg.Enabled = true
	case "false":
		cfg.Enabled = false
	default:
		return cfg, fmt.Errorf("RATE_LIMIT_ENABLED must be 'true' or 'false', got '%s'.", v)
	}

	var err error

	if cfg.PerMinute, err = positiveInt(getenv, "RATE_LIMIT_PER_MINUTE", DefaultPerMinute); err != nil {
		return cfg, err
	}

	if cfg.Burst, err = positiveInt(getenv, "RATE_LIMIT_BURST", DefaultBurst); err != nil {
		return cfg, err
	}

	if cfg.Exempt, err = parseExempt(getenv("RATE_LIMIT_EXEMPT")); err != nil {
		return cfg, err
	}

	return cfg, nil

}

func positiveInt(getenv func(string) string, key string, def int) (int, error) {

	v := strings.TrimSpace(getenv(key))

	if v == "" {
		return def, nil
	}

	n, err := strconv.Atoi(v)

	if err != nil {
		return 0, fmt.Errorf("%s environment variable is not an integer.", key)
	}

	if n <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero.", key)
	}

	return n, nil

}

/* parseExempt accepts a comma separated list of IPs and/or CIDRs, for example
   "10.0.5.12,192.168.0.0/24,::1". */

func parseExempt(v string) ([]netip.Prefix, error) {

	var out []netip.Prefix

	for _, item := range strings.Split(v, ",") {

		item = strings.TrimSpace(item)

		if item == "" {
			continue
		}

		if strings.Contains(item, "/") {

			p, err := netip.ParsePrefix(item)

			if err != nil {
				return nil, fmt.Errorf("RATE_LIMIT_EXEMPT: invalid CIDR '%s'.", item)
			}

			out = append(out, p.Masked())
			continue
		}

		a, err := netip.ParseAddr(item)

		if err != nil {
			return nil, fmt.Errorf("RATE_LIMIT_EXEMPT: invalid IP address '%s'.", item)
		}

		a = a.Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}

	return out, nil

}

/* String returns a one line summary suitable for a startup log message. */

func (c Config) String() string {

	if !c.Enabled {
		return "disabled"
	}

	s := fmt.Sprintf("%d/minute, burst %d", c.PerMinute, c.Burst)

	if len(c.Exempt) > 0 {
		ex := make([]string, len(c.Exempt))
		for i, p := range c.Exempt {
			ex[i] = p.String()
		}
		s += ", exempt: " + strings.Join(ex, ",")
	}

	return s

}

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type Limiter struct {
	cfg      Config
	mu       sync.Mutex
	limiters map[string]*ipLimiter
}

func New(cfg Config) *Limiter {
	return &Limiter{
		cfg:      cfg,
		limiters: make(map[string]*ipLimiter),
	}
}

func (rl *Limiter) isExempt(ip string) bool {

	if len(rl.cfg.Exempt) == 0 {
		return false
	}

	a, err := netip.ParseAddr(ip)

	if err != nil {
		return false
	}

	a = a.Unmap()

	for _, p := range rl.cfg.Exempt {
		if p.Contains(a) {
			return true
		}
	}

	return false

}

func (rl *Limiter) get(ip string) *rate.Limiter {

	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]

	if !exists {
		entry = &ipLimiter{
			limiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(rl.cfg.PerMinute)), rl.cfg.Burst),
		}
		rl.limiters[ip] = entry
	}

	entry.lastSeen = time.Now()
	return entry.limiter

}

func (rl *Limiter) clean() {

	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, entry := range rl.limiters {
		if time.Since(entry.lastSeen) > idleTimeout {
			delete(rl.limiters, ip)
		}
	}

}

/* StartCleanup periodically drops limiters for IPs that have gone quiet. */

func (rl *Limiter) StartCleanup(ctx context.Context) {

	if !rl.cfg.Enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.clean()
			case <-ctx.Done():
				return
			}
		}
	}()

}

/* Middleware rejects requests over the limit with 429 and a Retry-After header. */

func (rl *Limiter) Middleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		if !rl.cfg.Enabled {
			c.Next()
			return
		}

		ip := c.ClientIP()

		if rl.isExempt(ip) {
			c.Next()
			return
		}

		r := rl.get(ip).Reserve()

		if delay := r.Delay(); delay > 0 {
			r.Cancel()
			l.Logger(l.WARN, "Rate limit exceeded for %s on auth endpoint.", ip)
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(delay.Seconds()))))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}

		c.Next()

	}

}
