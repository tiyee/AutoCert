package deployer

import (
	"context"
	"testing"
	"time"

	"github.com/tiyee/AutoCert/internal/applicant"
	"github.com/tiyee/AutoCert/internal/edge"
)

type fakeEdgeAPI struct{ domain string }

func (f *fakeEdgeAPI) CheckTarget(ctx context.Context, domain string) error {
	f.domain = domain
	return ctx.Err()
}
func (f *fakeEdgeAPI) Deploy(ctx context.Context, domain, public, private string) error {
	f.domain = domain
	if public != "fake-cert" || private != "fake-key" {
		panic("wrong certificate")
	}
	return ctx.Err()
}
func (f *fakeEdgeAPI) Certificates(context.Context, string) ([]edge.Certificate, error) {
	return []edge.Certificate{{ID: "bound", Expires: time.Unix(1700000000, 0)}}, nil
}
func TestEdgeDeployerBridge(t *testing.T) {
	f := &fakeEdgeAPI{}
	d := &edgeDeployer{client: f, options: &Options{Domain: "example.invalid"}}
	d.SetCertificate(applicant.Certificate{Certificate: "fake-cert", PrivateKey: "fake-key"})
	if err := d.Deploy(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.domain != "example.invalid" {
		t.Fatal("wrong deployment domain")
	}
	certs, err := d.Search("example.invalid")
	if err != nil || len(certs) != 1 || certs[0].LastTime != 1700000000 {
		t.Fatal("expiry conversion failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if d.Deploy(ctx) == nil {
		t.Fatal("context lost")
	}
	for _, platform := range []string{"esa", "teo"} {
		if GetDeployer(platform) == nil {
			t.Fatal("missing deploy platform")
		}
	}
}

func TestTEORenewsEarliestBoundCertificate(t *testing.T) {
	d, err := NewTEO(WithAccessKeyId("fake-id"), WithAccessKeySecret("fake-key"))
	if err != nil {
		t.Fatal(err)
	}
	certs := []CertStat{{CertId: "later", LastTime: 200}, {CertId: "soon", LastTime: 100}}
	selected, ok := d.(*edgeDeployer).SelectRenewalCertificate(certs)
	if !ok || selected.CertId != "soon" {
		t.Fatal("expiring bound certificate ignored")
	}
	esa := &edgeDeployer{}
	selected, ok = esa.SelectRenewalCertificate(certs)
	if !ok || selected.CertId != "later" {
		t.Fatal("ESA historical certificate forced renewal")
	}
	if _, ok = d.(*edgeDeployer).SelectRenewalCertificate(nil); ok {
		t.Fatal("empty result accepted")
	}
}

func TestEdgePreflightBridge(t *testing.T) {
	f := &fakeEdgeAPI{}
	d := &edgeDeployer{client: f, options: &Options{Domain: "example.invalid"}}
	if err := d.Preflight(context.Background()); err != nil || f.domain != "example.invalid" {
		t.Fatal("target preflight not forwarded")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if d.Preflight(ctx) == nil {
		t.Fatal("preflight context lost")
	}
}
