package main

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

func TestTenantFromContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(r.Context(), UserKey, int64(7))
	ctx = context.WithValue(ctx, BusinessKey, int64(3))
	ctx = context.WithValue(ctx, CompanyKey, int64(11))
	ctx = context.WithValue(ctx, RoleKey, int64(store.ROLE_OWNER))

	got := tenantFrom(r.WithContext(ctx))

	want := tenant{UserID: 7, BusinessID: 3, CompanyID: 11, Role: store.ROLE_OWNER}
	if got != want {
		t.Fatalf("tenantFrom = %+v, want %+v", got, want)
	}
	if !got.IsOwner() {
		t.Fatal("role 1 must be business owner")
	}
}

func TestTenantFromContextEmpty(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)

	got := tenantFrom(r)

	if got.BusinessID != 0 || got.UserID != 0 {
		t.Fatalf("empty context must yield zero tenant, got %+v", got)
	}
	if got.IsOwner() {
		t.Fatal("zero tenant must not be an owner")
	}
}

func TestPickLoginUser(t *testing.T) {
	candidates := []store.User{
		{ID: 1, BusinessId: 10},
		{ID: 2, BusinessId: 20},
	}

	t.Run("bitta nomzod", func(t *testing.T) {
		user, err := pickLoginUser(candidates[:1], 0)
		if err != nil {
			t.Fatalf("pickLoginUser: %v", err)
		}
		if user.ID != 1 {
			t.Fatalf("got user %d, want 1", user.ID)
		}
	})

	t.Run("business_id bilan tanlash", func(t *testing.T) {
		user, err := pickLoginUser(candidates, 20)
		if err != nil {
			t.Fatalf("pickLoginUser: %v", err)
		}
		if user.ID != 2 {
			t.Fatalf("got user %d, want 2", user.ID)
		}
	})

	t.Run("noaniq business", func(t *testing.T) {
		if _, err := pickLoginUser(candidates, 0); err == nil {
			t.Fatal("bir nechta business uchun xato kutilgan edi")
		}
	})

	t.Run("boshqa businessga kirish mumkin emas", func(t *testing.T) {
		if _, err := pickLoginUser(candidates, 99); err == nil {
			t.Fatal("mos kelmaydigan business uchun xato kutilgan edi")
		}
	})

	t.Run("nomzod yo'q", func(t *testing.T) {
		if _, err := pickLoginUser(nil, 0); err == nil {
			t.Fatal("bo'sh ro'yxat uchun xato kutilgan edi")
		}
	})
}

func TestAnyToInt64(t *testing.T) {
	cases := []struct {
		in   any
		want int64
	}{
		{int64(5), 5},
		{int(6), 6},
		{float64(7), 7}, // JSON raqamlari float64 bo'lib keladi
		{"8", 8},
		{"abc", 0},
		{nil, 0},
	}

	for _, c := range cases {
		if got := anyToInt64(c.in); got != c.want {
			t.Fatalf("anyToInt64(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}
