package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlatformAdminMiddleware(t *testing.T) {
	app := &application{}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("passed"))
	})

	cases := []struct {
		name       string
		configured string
		header     string
		wantStatus int
		wantBody   bool
	}{
		{
			name:       "kalit sozlanmagan — endpoint o'chirilgan",
			configured: "",
			header:     "anything",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "kalit yuborilmagan",
			configured: "s3cret",
			header:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "noto'g'ri kalit",
			configured: "s3cret",
			header:     "wrong",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "to'g'ri kalit",
			configured: "s3cret",
			header:     "s3cret",
			wantStatus: http.StatusOK,
			wantBody:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ADMIN_API_KEY", tc.configured)

			r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/companies", nil)
			if tc.header != "" {
				r.Header.Set("X-Admin-Key", tc.header)
			}
			w := httptest.NewRecorder()

			app.PlatformAdminMiddleware(next).ServeHTTP(w, r)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body.String())
			}
			if tc.wantBody && w.Body.String() != "passed" {
				t.Fatalf("handler was not reached, body: %s", w.Body.String())
			}
			if !tc.wantBody && w.Body.String() == "passed" {
				t.Fatal("handler must not be reached")
			}
		})
	}
}
