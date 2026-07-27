package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

// TestCompanyRoutesAreMounted — kompaniya CRUD yo'llari router'da bormi.
// Tokensiz so'rov 401 qaytaradi (yo'l topildi), yo'l umuman bo'lmasa 404.
func TestCompanyRoutesAreMounted(t *testing.T) {
	mux := (&application{}).mount()

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/user/companies"},
		{http.MethodPut, "/api/v1/user/companies/12"},
		{http.MethodDelete, "/api/v1/user/companies/12"},
		{http.MethodGet, "/api/v1/user/companies/all"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, r)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, http.StatusUnauthorized, w.Body.String())
			}
		})
	}
}

func TestMergeCompanyPayload(t *testing.T) {
	current := &store.Company{
		ID:         7,
		Name:       "Chilonzor filiali",
		Details:    "eski izoh",
		Password:   "eski-parol",
		BusinessID: 3,
		CreatedAt:  "2026-01-01",
	}

	cases := []struct {
		name         string
		payload      CompanyPayload
		wantName     string
		wantDetails  string
		wantPassword string
	}{
		{
			name:         "bo'sh payload eski qiymatlarni saqlaydi",
			payload:      CompanyPayload{},
			wantName:     "Chilonzor filiali",
			wantDetails:  "",
			wantPassword: "eski-parol",
		},
		{
			name:         "faqat probel bo'lgan nom o'zgartirmaydi",
			payload:      CompanyPayload{Name: "   "},
			wantName:     "Chilonzor filiali",
			wantDetails:  "",
			wantPassword: "eski-parol",
		},
		{
			name:         "to'liq yangilanish",
			payload:      CompanyPayload{Name: " Yunusobod ", Details: "yangi izoh", Password: "yangi-parol"},
			wantName:     "Yunusobod",
			wantDetails:  "yangi izoh",
			wantPassword: "yangi-parol",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeCompanyPayload(current, tc.payload)

			if got.Name != tc.wantName {
				t.Fatalf("Name = %q, want %q", got.Name, tc.wantName)
			}
			if got.Details != tc.wantDetails {
				t.Fatalf("Details = %q, want %q", got.Details, tc.wantDetails)
			}
			if got.Password != tc.wantPassword {
				t.Fatalf("Password = %q, want %q", got.Password, tc.wantPassword)
			}
			if got.ID != current.ID || got.BusinessID != current.BusinessID {
				t.Fatalf("id/business o'zgardi: %+v", got)
			}
		})
	}
}
