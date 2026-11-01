package edge

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func response[T any](value string) *T {
	var v T
	if err := json.Unmarshal([]byte(value), &v); err != nil {
		panic(err)
	}
	return &v
}
func testCertificate(t *testing.T, names ...string) (string, string, time.Time) {
	t.Helper()
	return testCertificateValidity(t, time.Now().Add(-time.Hour), time.Now().Add(60*24*time.Hour).Truncate(time.Second), names...)
}
func testCertificateValidity(t *testing.T, start, end time.Time, names ...string) (string, string, time.Time) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: names, NotBefore: start, NotAfter: end}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	pk, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pk})), end
}
func TestSelectSite(t *testing.T) {
	sites := []Site{{ID: "parent", Name: "example.com", DNS: true}, {ID: "duplicate", Name: "example.com", DNS: true}, {ID: "sub", Name: "sub.example.com", DNS: true}}
	s, err := SelectSite(sites, "_acme-challenge.sub.example.com.", "", true)
	if err != nil || s.ID != "sub" {
		t.Fatalf("%v %v", s, err)
	}
	if _, err := SelectSite(sites, "www.example.com", "", false); err == nil {
		t.Fatal("ambiguity accepted")
	}
	if _, err := SelectSite(sites, "notexample.com", "", false); err == nil {
		t.Fatal("label boundary ignored")
	}
	s, err = SelectSite(sites, "*.example.com", "parent", false)
	if err != nil || s.ID != "parent" {
		t.Fatal(err)
	}
	if _, err := SelectSite(sites, "example.com", "sub", false); err == nil {
		t.Fatal("mismatched ID accepted")
	}
	if _, err := SelectSite([]Site{{ID: "x", Name: "example.com"}}, "example.com", "", true); err == nil {
		t.Fatal("CNAME DNS accepted")
	}
	s, err = SelectSite([]Site{{ID: "idn", Name: "例子.公司.cn"}}, "_acme-challenge.例子.公司.cn", "", false)
	if err != nil || s.ID != "idn" {
		t.Fatal(err)
	}
}
func TestCertificateValidation(t *testing.T) {
	cert, key, _ := testCertificate(t, "*.example.com", "example.com")
	for _, domain := range []string{"*.example.com", "www.example.com", "example.com"} {
		if _, err := validateDeployment(cert, key, domain); err != nil {
			t.Fatal(err)
		}
	}
	for _, domain := range []string{"a.b.example.com", "other.com"} {
		if _, err := ParseCertificate(cert, domain); err == nil {
			t.Fatal("wrong domain accepted")
		}
	}
	if _, err := validateDeployment(cert, "invalid", "example.com"); err == nil {
		t.Fatal("invalid key accepted")
	}
	if _, err := ParseCertificate("invalid", "example.com"); err == nil {
		t.Fatal("invalid PEM accepted")
	}
	if err := apiError("upload", &testError{}); strings.Contains(err.Error(), "private-content") {
		t.Fatal("request leaked")
	}
}

type testError struct{}

func (*testError) Error() string { return "private-content" }

func TestSelectSiteSkipsDomainlessAndUnrelatedSites(t *testing.T) {
	sites := []Site{{ID: "pages"}, {ID: "broken", Name: "bad domain"}, {ID: "selected", Name: "example.com", DNS: true}}
	for _, id := range []string{"", "selected"} {
		s, err := SelectSite(sites, "www.example.com", id, false)
		if err != nil || s.ID != "selected" {
			t.Fatal("unrelated site blocked selection", err)
		}
	}
	if _, err := SelectSite(sites, "www.example.com", "pages", false); err == nil {
		t.Fatal("explicit domainless site accepted")
	}
}
func TestCertificateValidityAndNames(t *testing.T) {
	for _, span := range [][2]time.Time{{time.Now().Add(time.Hour), time.Now().Add(48 * time.Hour)}, {time.Now().Add(-48 * time.Hour), time.Now().Add(-time.Hour)}} {
		cert, key, _ := testCertificateValidity(t, span[0], span[1], "example.com")
		if _, err := validateDeployment(cert, key, "example.com"); err == nil {
			t.Fatal("invalid certificate validity accepted")
		}
	}
	a := certificateName("www.example.com", []string{"www.example.com", "EXAMPLE.COM."})
	b := certificateName("www.example.com", []string{"example.com", "www.example.com", "www.example.com"})
	c := certificateName("www.example.com", []string{"www.example.com"})
	if a != b || a == c {
		t.Fatal("certificate name is not scoped to canonical SAN set")
	}
	if len(certificateName(strings.Repeat("a", 240), []string{"example.com"})) > 128 {
		t.Fatal("ESA certificate name too long")
	}
}
