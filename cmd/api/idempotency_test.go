package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestDedupConcurrentIdenticalRequests — foydalanuvchi stsenariysi:
// 1 soniya ichida bir xil body bilan 3 ta so'rov yuborilsa,
// faqat bittasi handlerga yetib borishi va qolgan ikkitasi 409 olishi kerak.
func TestDedupConcurrentIdenticalRequests(t *testing.T) {
	app := &application{dedup: newIdempotencyGuard(10 * time.Second)}

	var handlerHits int64
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&handlerHits, 1)
		// Real handlerni taqlid qilamiz: ozgina ushlab turadi (DB transaction).
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	mw := app.DedupCreateMiddleware(slow)
	body := []byte(`{"received_money":100,"received_currency":"USD","user_id":7}`)

	const n = 3
	var wg sync.WaitGroup
	statuses := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/user/exchanges", bytes.NewReader(body))
			rec := httptest.NewRecorder()
			mw.ServeHTTP(rec, req)
			statuses[idx] = rec.Code
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt64(&handlerHits); got != 1 {
		t.Fatalf("handler %d marta ishladi, kutilgan 1 (qolganlari 409 bo'lishi kerak edi)", got)
	}

	var ok, dup int
	for _, s := range statuses {
		switch s {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			dup++
		default:
			t.Fatalf("kutilmagan status: %d", s)
		}
	}
	if ok != 1 || dup != n-1 {
		t.Fatalf("statuslar noto'g'ri: ok=%d dup=%d (kutilgan ok=1 dup=%d)", ok, dup, n-1)
	}
}

// TestDedupReleasesOnFailure — birinchi so'rov xato (>=400) qaytarsa,
// kalit bo'shaydi va foydalanuvchi haqiqatan qayta yubora oladi.
func TestDedupReleasesOnFailure(t *testing.T) {
	app := &application{dedup: newIdempotencyGuard(10 * time.Second)}

	var hits int64
	failing := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	})
	mw := app.DedupCreateMiddleware(failing)
	body := []byte(`{"a":1}`)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("urinish %d: status %d, kutilgan 500", i, rec.Code)
		}
	}
	if hits != 2 {
		t.Fatalf("handler %d marta ishladi, kutilgan 2 (xatodan keyin qayta yuborilishi mumkin)", hits)
	}
}
