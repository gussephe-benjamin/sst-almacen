package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// loginLimiter bloquea temporalmente un DNI o una IP tras varios intentos fallidos.
type loginLimiter struct {
	mu    sync.Mutex
	fails map[string][]time.Time
}

const (
	maxFailsDNI = 5  // intentos fallidos por DNI
	maxFailsIP  = 30 // por IP: alto, porque todo el almacén puede salir por la misma IP
	failWindow  = 15 * time.Minute
)

func limitFor(key string) int {
	if len(key) > 3 && key[:3] == "ip:" {
		return maxFailsIP
	}
	return maxFailsDNI
}

func newLoginLimiter() *loginLimiter { return &loginLimiter{fails: map[string][]time.Time{}} }

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *loginLimiter) recent(key string) []time.Time {
	var out []time.Time
	for _, t := range l.fails[key] {
		if time.Since(t) < failWindow {
			out = append(out, t)
		}
	}
	l.fails[key] = out
	return out
}

func (l *loginLimiter) blocked(keys ...string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		if len(l.recent(k)) >= limitFor(k) {
			return true
		}
	}
	return false
}

func (l *loginLimiter) fail(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		l.fails[k] = append(l.recent(k), time.Now())
	}
}

func (l *loginLimiter) reset(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		delete(l.fails, k)
	}
}
