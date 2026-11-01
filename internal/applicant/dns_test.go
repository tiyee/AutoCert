package applicant

import (
	"errors"
	"testing"
)

func TestOwnedDNSLifecycle(t *testing.T) {
	t.Setenv("LEGO_DISABLE_CNAME_SUPPORT", "true")
	var created, deleted int
	p := &ownedDNSProvider{create: func(fqdn, value string) (recordHandle, error) {
		created++
		return recordHandle{zone: "example.invalid", id: value}, nil
	}, remove: func(recordHandle) error { deleted++; return nil }}
	if err := p.Present("example.invalid", "", "first"); err != nil {
		t.Fatal(err)
	}
	if err := p.Present("EXAMPLE.INVALID.", "", "first"); err != nil {
		t.Fatal(err)
	}
	if err := p.Present("example.invalid", "", "second"); err != nil {
		t.Fatal(err)
	}
	if created != 2 {
		t.Fatalf("created %d", created)
	}
	if err := p.CleanUp("example.invalid", "", "first"); err != nil {
		t.Fatal(err)
	}
	if deleted != 0 {
		t.Fatal("deleted shared record too soon")
	}
	if err := p.CleanUp("example.invalid", "", "second"); err != nil {
		t.Fatal(err)
	}
	if len(p.records) != 1 || deleted != 1 {
		t.Fatal("cleanup removed another challenge")
	}
	if err := p.CleanUp("example.invalid", "", "first"); err != nil {
		t.Fatal(err)
	}
	if err := p.CleanUp("example.invalid", "", "first"); err != nil {
		t.Fatal(err)
	}
	if deleted != 2 || len(p.records) != 0 {
		t.Fatal("cleanup is not idempotent")
	}
}
func TestOwnedDNSFailure(t *testing.T) {
	t.Setenv("LEGO_DISABLE_CNAME_SUPPORT", "true")
	deleted := 0
	p := &ownedDNSProvider{create: func(string, string) (recordHandle, error) { return recordHandle{}, errors.New("write failed") }, remove: func(recordHandle) error { deleted++; return nil }}
	if p.Present("example.invalid", "", "first") == nil {
		t.Fatal("expected creation error")
	}
	if err := p.CleanUp("example.invalid", "", "first"); err != nil {
		t.Fatal(err)
	}
	if deleted != 0 {
		t.Fatal("deleted a record after failed creation")
	}
	p.create = func(string, string) (recordHandle, error) { return recordHandle{id: "own-record"}, nil }
	if err := p.Present("example.invalid", "", "first"); err != nil {
		t.Fatal(err)
	}
	p.remove = func(recordHandle) error { return errors.New("delete failed") }
	if p.CleanUp("example.invalid", "", "first") == nil || len(p.records) != 1 {
		t.Fatal("failed deletion lost ownership")
	}
	p.remove = func(recordHandle) error { deleted++; return nil }
	if err := p.CleanUp("example.invalid", "", "first"); err != nil {
		t.Fatal(err)
	}
	if deleted != 1 || len(p.records) != 0 {
		t.Fatal("retry failed")
	}
}

func TestCertificateDomains(t *testing.T) {
	for _, domain := range []string{"example.com", "*.example.com", "*.example.com.cn", "*.sub.example.com"} {
		names := certificateDomains(domain)
		if domain[0] == '*' {
			if len(names) != 2 || names[1] != domain[2:] {
				t.Fatal(names)
			}
		} else if len(names) != 1 {
			t.Fatal(names)
		}
	}
}
