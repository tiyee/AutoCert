package deployer

import (
	"testing"
	"time"
)

func TestRenewalBoundary(t *testing.T) {
	now := time.Unix(1700000000, 0)
	for _, tc := range []struct {
		offset int64
		want   bool
	}{{-1, true}, {0, true}, {864000 - 1, true}, {864000, true}, {864001, false}} {
		if got := (CertStat{LastTime: now.Unix() + tc.offset}).renewableAt(now); got != tc.want {
			t.Errorf("offset %d: %v", tc.offset, got)
		}
	}
	latest, ok := LatestCertificate([]CertStat{{LastTime: 10}, {LastTime: 30}, {LastTime: 20}})
	if !ok || latest.LastTime != 30 {
		t.Fatal("selected wrong certificate")
	}
	if _, ok := LatestCertificate(nil); ok {
		t.Fatal("empty list")
	}
}
func TestTencentExpiry(t *testing.T) {
	got, err := parseTencentExpiry("2026-11-01 08:00:00")
	if err != nil {
		t.Fatal(err)
	}
	if got.UTC().Format(time.RFC3339) != "2026-11-01T00:00:00Z" {
		t.Fatal(got)
	}
	for _, value := range []string{"", "invalid"} {
		if _, err := parseTencentExpiry(value); err == nil {
			t.Fatal("accepted invalid expiry")
		}
	}
}
func TestCoversDomain(t *testing.T) {
	for _, tc := range []struct {
		pattern, domain string
		want            bool
	}{
		{"*.example.com", "www.example.com", true}, {"*.example.com", "example.com", false}, {"*.example.com", "a.b.example.com", false}, {"*.example.com", "*.example.com", true}, {"www.example.com", "*.example.com", false}, {"EXAMPLE.COM.", "example.com", true}, {"badexample.com", "example.com", false}, {"", "", false},
	} {
		if got := coversDomain(tc.pattern, tc.domain); got != tc.want {
			t.Errorf("%s covers %s: %v", tc.pattern, tc.domain, got)
		}
	}
}
