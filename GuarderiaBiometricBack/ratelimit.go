package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimiter implementa un limitador simple de "ventana fija" por IP, en memoria.
// Es suficiente para una única instancia del backend; si en el futuro se despliega
// con más de una réplica, esto debe moverse a un almacén compartido (ej. Redis).
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitorEntry
	limit    int
	window   time.Duration
}

type visitorEntry struct {
	count       int
	windowStart time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitorEntry),
		limit:    limit,
		window:   window,
	}
	go rl.cleanupLoop()
	return rl
}

// cleanupLoop libera periódicamente entradas de IPs que ya no tienen actividad
// reciente, para que el mapa no crezca indefinidamente.
func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		rl.mu.Lock()
		for key, entry := range rl.visitors {
			if now.Sub(entry.windowStart) > rl.window {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware bloquea con 429 cuando una misma IP excede "limit" peticiones
// dentro de la ventana configurada.
func (rl *rateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now()

		rl.mu.Lock()
		entry, ok := rl.visitors[key]
		if !ok || now.Sub(entry.windowStart) > rl.window {
			entry = &visitorEntry{count: 0, windowStart: now}
			rl.visitors[key] = entry
		}
		entry.count++
		count := entry.count
		rl.mu.Unlock()

		if count > rl.limit {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Demasiados intentos. Intenta de nuevo en unos minutos."})
			c.Abort()
			return
		}

		c.Next()
	}
}
