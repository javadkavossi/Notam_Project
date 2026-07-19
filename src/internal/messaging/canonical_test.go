package messaging

import "testing"

func TestCanonicalKey(t *testing.T) {
	cases := []struct {
		loc, series, want string
	}{
		{"OIII", "A0046/26", "OIII|A0046/26"},
		{" oiii ", " a0046/26 ", "OIII|A0046/26"}, // trim + uppercase
		{"OIII", "", ""},                          // بدون series → خالی (fallback در repo)
		{"", "A0046/26", "|A0046/26"},
	}
	for _, c := range cases {
		if got := CanonicalKey(c.loc, c.series); got != c.want {
			t.Errorf("CanonicalKey(%q,%q)=%q، انتظار %q", c.loc, c.series, got, c.want)
		}
	}
}
