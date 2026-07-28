package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

// fakeUserStore — faqat switch oqimi uchun kerakli metodlar ishlaydi, qolganlari
// testda chaqirilmagani uchun bo'sh qiymat qaytaradi.
type fakeUserStore struct {
	users []store.User
}

func (f *fakeUserStore) LoginCandidates(_ context.Context, phone, password string) ([]store.User, error) {
	var out []store.User
	for _, u := range f.users {
		if u.Phone == phone && u.Password == password {
			out = append(out, u)
		}
	}
	return out, nil
}

func (f *fakeUserStore) GetById(_ context.Context, id *int64) (*store.User, error) {
	for i := range f.users {
		if f.users[i].ID == *id {
			user := f.users[i]
			return &user, nil
		}
	}
	return nil, errors.New("USER NOT FOUND")
}

func (f *fakeUserStore) Create(context.Context, *store.User) error { return nil }
func (f *fakeUserStore) Update(context.Context, *store.User) error { return nil }
func (f *fakeUserStore) ListByBusiness(context.Context, int64) ([]store.User, error) {
	return nil, nil
}
func (f *fakeUserStore) ListByCompany(context.Context, int64, int64) ([]store.User, error) {
	return nil, nil
}
func (f *fakeUserStore) GetByIdInBusiness(context.Context, int64, int64) (*store.User, error) {
	return nil, nil
}
func (f *fakeUserStore) Delete(context.Context, *int64, int64) error { return nil }

type fakeBusinessStore struct {
	businesses map[int64]store.Business
}

func (f *fakeBusinessStore) ListByIDs(_ context.Context, ids []int64) (map[int64]store.Business, error) {
	out := make(map[int64]store.Business, len(ids))
	for _, id := range ids {
		if b, ok := f.businesses[id]; ok {
			out[id] = b
		}
	}
	return out, nil
}

func (f *fakeBusinessStore) Create(context.Context, *store.Business) error { return nil }
func (f *fakeBusinessStore) GetById(context.Context, int64) (*store.Business, error) {
	return nil, nil
}
func (f *fakeBusinessStore) Update(context.Context, *store.Business) error { return nil }
func (f *fakeBusinessStore) SetStatus(context.Context, int64, int64) error { return nil }
func (f *fakeBusinessStore) Delete(context.Context, int64) error           { return nil }

// switchTestApp — bir telefon (90 123 45 67) ikkita businessda: 10 va 20.
// 30-business boshqa odamniki (telefon boshqa).
func switchTestApp() *application {
	users := &fakeUserStore{users: []store.User{
		{ID: 1, Phone: "901234567", Password: "p", Username: "Ali", BusinessId: 10, CompanyId: 100, Role: store.ROLE_OWNER},
		{ID: 2, Phone: "901234567", Password: "p", Username: "Ali", BusinessId: 20, CompanyId: 200, Role: store.ROLE_STAFF},
		{ID: 3, Phone: "907777777", Password: "p", Username: "Vali", BusinessId: 30, CompanyId: 300, Role: store.ROLE_OWNER},
		// Parol boshqa: bir xil telefon bo'lsa ham switch ro'yxatiga tushmaydi.
		{ID: 4, Phone: "901234567", Password: "boshqa", Username: "Ali", BusinessId: 40, CompanyId: 400, Role: store.ROLE_STAFF},
	}}

	return &application{
		store: store.Storage{
			Users: users,
			Businesses: &fakeBusinessStore{businesses: map[int64]store.Business{
				10: {ID: 10, Name: "Business A", Status: store.BUSINESS_ACTIVE},
				20: {ID: 20, Name: "Business B", Status: store.BUSINESS_ACTIVE},
				30: {ID: 30, Name: "Business C", Status: store.BUSINESS_ACTIVE},
				40: {ID: 40, Name: "Business D", Status: store.BUSINESS_ACTIVE},
			}},
		},
	}
}

func switchRequest(userID int64, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users/switch-business", strings.NewReader(body))
	return r.WithContext(context.WithValue(r.Context(), UserKey, userID))
}

type switchResponse struct {
	Token string     `json:"token"`
	User  store.User `json:"user"`
	// businesses ro'yxati mijozga almashish uchun ko'rsatiladi.
	Businesses []businessMembership `json:"businesses"`
}

func decodeSwitch(t *testing.T, w *httptest.ResponseRecorder) switchResponse {
	t.Helper()

	var envelope struct {
		Data switchResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("javobni o'qib bo'lmadi: %v (body: %s)", err, w.Body.String())
	}
	return envelope.Data
}

// Parol qayta terilmasdan ikkinchi businessga o'tish ishlaydi.
func TestSwitchBusinessToSibling(t *testing.T) {
	app := switchTestApp()
	w := httptest.NewRecorder()

	app.SwitchBusinessHandler(w, switchRequest(1, `{"business_id": 20}`))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	got := decodeSwitch(t, w)
	if got.User.ID != 2 || got.User.BusinessId != 20 {
		t.Fatalf("user = %d/business %d, want 2/20", got.User.ID, got.User.BusinessId)
	}

	// Yangi token maqsadli business useriga bog'langan bo'lishi shart.
	claims, err := parseToken(got.Token, accessTokenType)
	if err != nil {
		t.Fatalf("parseToken: %v", err)
	}
	id, err := claims.userID()
	if err != nil {
		t.Fatalf("userID: %v", err)
	}
	if id != 2 {
		t.Fatalf("token userID = %d, want 2", id)
	}
	if claims.BusinessID != 20 {
		t.Fatalf("token businessID = %d, want 20", claims.BusinessID)
	}
}

// Boshqa odamning businessiga ham, parol mos kelmaydigan businessga ham o'tib bo'lmaydi.
func TestSwitchBusinessDenied(t *testing.T) {
	cases := []struct {
		name       string
		businessID int64
	}{
		{"boshqa odamning businessi", 30},
		{"parol boshqa bo'lgan business", 40},
		{"umuman yo'q business", 99},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := switchTestApp()
			w := httptest.NewRecorder()

			app.SwitchBusinessHandler(w, switchRequest(1, `{"business_id": `+strconv.FormatInt(tc.businessID, 10)+`}`))

			if w.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 (body: %s)", w.Code, w.Body.String())
			}
		})
	}
}

// Javobdagi businesses ro'yxatida faqat parol ochadigan businesslar bo'ladi.
func TestSwitchBusinessMembershipList(t *testing.T) {
	app := switchTestApp()
	w := httptest.NewRecorder()

	app.ListMyBusinessesHandler(w, switchRequest(1, ""))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	var envelope struct {
		Data []businessMembership `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("javobni o'qib bo'lmadi: %v (body: %s)", err, w.Body.String())
	}

	memberships := envelope.Data
	if len(memberships) != 2 {
		t.Fatalf("a'zoliklar soni = %d, want 2 (%+v)", len(memberships), memberships)
	}
	if memberships[0].BusinessID != 10 || memberships[1].BusinessID != 20 {
		t.Fatalf("businesslar = %d,%d want 10,20", memberships[0].BusinessID, memberships[1].BusinessID)
	}
	if memberships[0].BusinessName != "Business A" {
		t.Fatalf("nom = %q, want %q", memberships[0].BusinessName, "Business A")
	}
	if memberships[1].Role != store.ROLE_STAFF {
		t.Fatalf("rol = %d, want %d", memberships[1].Role, store.ROLE_STAFF)
	}
}

// Switch va ro'yxat yo'llari router'ga ulanganmi (tokensiz 401, 404 emas).
func TestSwitchBusinessRoutesAreMounted(t *testing.T) {
	mux := (&application{}).mount()

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/users/switch-business"},
		{http.MethodGet, "/api/v1/users/businesses"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, r)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body: %s)", w.Code, w.Body.String())
			}
		})
	}
}
