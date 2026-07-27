package store

import "testing"

func TestBusinessSettingsIsThreeStage(t *testing.T) {
	cases := []struct {
		name string
		flow int64
		want bool
	}{
		{"standart oqim", TRANSACTION_FLOW_SIMPLE, false},
		{"3 bosqichli oqim", TRANSACTION_FLOW_THREE_STAGE, true},
		{"noma'lum qiymat oddiy hisoblanadi", 99, false},
		{"nol qiymat oddiy hisoblanadi", 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			settings := BusinessSettings{TransactionFlow: tc.flow}
			if got := settings.IsThreeStage(); got != tc.want {
				t.Fatalf("IsThreeStage() = %v, want %v (flow=%d)", got, tc.want, tc.flow)
			}
		})
	}
}

func TestValidTransactionFlow(t *testing.T) {
	valid := []int64{TRANSACTION_FLOW_SIMPLE, TRANSACTION_FLOW_THREE_STAGE}
	for _, flow := range valid {
		if !ValidTransactionFlow(flow) {
			t.Fatalf("flow %d qabul qilinishi kerak", flow)
		}
	}

	for _, flow := range []int64{0, 3, -1, 100} {
		if ValidTransactionFlow(flow) {
			t.Fatalf("flow %d rad etilishi kerak", flow)
		}
	}
}

func TestTransactionStatusesAreDistinct(t *testing.T) {
	// STATUS_ACCEPTED oxiriga qo'shildi: mavjud yozuvlardagi 1/2/3 o'zgarmasligi shart.
	seen := map[int]string{
		STATUS_CREATED:   "created",
		STATUS_COMPLETED: "completed",
		STATUS_ARCHIVED:  "archived",
		STATUS_ACCEPTED:  "accepted",
	}
	if len(seen) != 4 {
		t.Fatalf("status qiymatlari takrorlanmasligi kerak: %v", seen)
	}
	if STATUS_ACCEPTED != 4 {
		t.Fatalf("STATUS_ACCEPTED = %d, want 4", STATUS_ACCEPTED)
	}
}
