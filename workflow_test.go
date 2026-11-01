package AutoCert

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tiyee/AutoCert/internal/applicant"
	"github.com/tiyee/AutoCert/internal/config"
	"github.com/tiyee/AutoCert/internal/deployer"
)

type fakeApplicant struct {
	calls int
	err   error
}

func (f *fakeApplicant) Apply() (*applicant.Certificate, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &applicant.Certificate{}, nil
}

type fakeDeployer struct {
	certs                []deployer.CertStat
	searchErr, deployErr error
	deployed             int
}

func (f *fakeDeployer) Search(string) ([]deployer.CertStat, error) { return f.certs, f.searchErr }
func (f *fakeDeployer) SetCertificate(applicant.Certificate)       {}
func (f *fakeDeployer) Deploy(context.Context) error               { f.deployed++; return f.deployErr }
func setupFlow(t *testing.T) (config.Config, *fakeApplicant, *fakeDeployer) {
	a := &fakeApplicant{}
	d := &fakeDeployer{}
	platform := "test-" + t.Name()
	applicant.RegisterApplicant(platform, func(...applicant.Option) applicant.IApplicant { return a })
	deployer.RegisterDeployer(platform, func(...deployer.Option) (deployer.IDeployer, error) { return d, nil })
	return config.Config{Platform: platform, Domain: "example.invalid"}, a, d
}
func TestRenewChoosesLatest(t *testing.T) {
	cfg, a, d := setupFlow(t)
	d.certs = []deployer.CertStat{{LastTime: 1}, {LastTime: time.Now().Add(30 * 24 * time.Hour).Unix()}}
	if err := Renew(cfg); err != nil {
		t.Fatal(err)
	}
	if a.calls != 0 || d.deployed != 0 {
		t.Fatal("renewed despite valid certificate")
	}
	d.certs = []deployer.CertStat{{LastTime: time.Now().Add(time.Hour).Unix()}}
	if err := Renew(cfg); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || d.deployed != 1 {
		t.Fatal("renewal not deployed")
	}
}
func TestFlowErrors(t *testing.T) {
	cfg, a, d := setupFlow(t)
	if Renew(cfg) == nil {
		t.Fatal("empty search accepted")
	}
	d.searchErr = errors.New("query failed")
	if Renew(cfg) == nil {
		t.Fatal("query failure lost")
	}
	d.searchErr = nil
	d.certs = []deployer.CertStat{{LastTime: 1}}
	a.err = errors.New("apply failed")
	if Renew(cfg) == nil || d.deployed != 0 {
		t.Fatal("application failure lost")
	}
	a.err = nil
	d.deployErr = errors.New("deploy failed")
	if CertOnly(cfg) == nil {
		t.Fatal("deployment failure lost")
	}
}
func TestPreflightBeforeIssuing(t *testing.T) {
	cfg, a, _ := setupFlow(t)
	cfg.CDNPlatform = "unsupported"
	if CertOnly(cfg) == nil || a.calls != 0 {
		t.Fatal("issued before validating deploy platform")
	}
}

type preflightDeployer struct {
	*fakeDeployer
	err    error
	checks int
}

func (f *preflightDeployer) Preflight(context.Context) error { f.checks++; return f.err }
func (f *preflightDeployer) SelectRenewalCertificate(certs []deployer.CertStat) (deployer.CertStat, bool) {
	if len(certs) == 0 {
		return deployer.CertStat{}, false
	}
	return certs[0], true
}
func TestTargetPreflightBeforeACME(t *testing.T) {
	cfg, a, d := setupFlow(t)
	checked := &preflightDeployer{fakeDeployer: d, err: errors.New("target missing")}
	deployer.RegisterDeployer(cfg.Platform, func(...deployer.Option) (deployer.IDeployer, error) { return checked, nil })
	if CertOnly(cfg) == nil || a.calls != 0 || checked.checks != 1 {
		t.Fatal("ACME called before target check")
	}
	checked.err = nil
	d.certs = []deployer.CertStat{{LastTime: time.Now().Add(time.Hour).Unix()}, {LastTime: time.Now().Add(40 * 24 * time.Hour).Unix()}}
	if err := Renew(cfg); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || d.deployed != 1 {
		t.Fatal("platform renewal selection ignored")
	}
}
