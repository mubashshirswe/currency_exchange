package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// idempotencyGuard — bir xil so'rovlarni qisqa oyna ichida (masalan 10s)
// faqat bir marta o'tkazadi. Jarayon ichida atomar (mutex bilan).
type idempotencyGuard struct {
	mu     sync.Mutex
	seen   map[string]time.Time // key -> expiry
	ttl    time.Duration
	lastGC time.Time
}

func newIdempotencyGuard(ttl time.Duration) *idempotencyGuard {
	return &idempotencyGuard{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}
}

// reserve — kalit yangi bo'lsa true (band qilindi), TTL ichida takror bo'lsa false.
func (g *idempotencyGuard) reserve(key string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Vaqti o'tgan kalitlarni vaqti-vaqti bilan tozalash.
	if now.Sub(g.lastGC) >= g.ttl {
		for k, exp := range g.seen {
			if now.After(exp) {
				delete(g.seen, k)
			}
		}
		g.lastGC = now
	}

	if exp, ok := g.seen[key]; ok && now.Before(exp) {
		return false
	}
	g.seen[key] = now.Add(g.ttl)
	return true
}

// release — operatsiya muvaffaqiyatsiz bo'lganda kalitni bo'shatadi,
// shunda foydalanuvchi haqiqatan qayta yuborishi mumkin.
func (g *idempotencyGuard) release(key string) {
	g.mu.Lock()
	delete(g.seen, key)
	g.mu.Unlock()
}

// statusRecorder — handler qaytargan HTTP statusni ushlab qoladi.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rec *statusRecorder) WriteHeader(code int) {
	if !rec.wroteHeader {
		rec.status = code
		rec.wroteHeader = true
	}
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *statusRecorder) Write(b []byte) (int, error) {
	if !rec.wroteHeader {
		rec.wroteHeader = true
	}
	return rec.ResponseWriter.Write(b)
}

// DedupCreateMiddleware — bir xil (user + method + path + body) so'rov TTL oynasida
// faqat bir marta bajariladi. Qolganlari 409 bilan bekor qilinadi.
func (app *application) DedupCreateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
		// Body'ni handler uchun qayta tiklash.
		r.Body = io.NopCloser(bytes.NewReader(body))

		userID, _ := r.Context().Value(UserKey).(int64)
		sum := sha256.Sum256(body)
		key := fmt.Sprintf("%d|%s|%s|%s", userID, r.Method, r.URL.Path, hex.EncodeToString(sum[:]))

		if !app.dedup.reserve(key, time.Now()) {
			app.duplicateRequestResponse(w, r)
			return
		}

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		// Operatsiya muvaffaqiyatsiz tugasa — kalitni bo'shatamiz.
		if rec.status >= http.StatusBadRequest {
			app.dedup.release(key)
		}
	})
}

func (app *application) duplicateRequestResponse(w http.ResponseWriter, r *http.Request) {
	log.Printf("duplicate request ignored: %s path: %s", r.Method, r.URL.Path)
	writeError(w, http.StatusConflict, "DUPLICATE_REQUEST")
}
