package AutoCert

import (
	"github.com/tiyee/AutoCert/internal/applicant"
	"github.com/tiyee/AutoCert/internal/config"
	"github.com/tiyee/AutoCert/internal/deployer"
	"os"
	"testing"
)

func TestCLISeparateSTSCredentials(t *testing.T) {
	oldArgs, oldCfg := os.Args, Cfg
	oldDomain, oldEmail, oldDNS, oldCDN, oldRenew, oldVersion := *domain, *email, *dnsSecurityToken, *cdnSecurityToken, *renew, *version
	defer func() {
		os.Args = oldArgs
		Cfg = oldCfg
		*domain = oldDomain
		*email = oldEmail
		*dnsSecurityToken = oldDNS
		*cdnSecurityToken = oldCDN
		*renew = oldRenew
		*version = oldVersion
	}()
	os.Args = []string{"autocert"}
	*domain = "example.invalid"
	*email = "test@example.invalid"
	*version = false
	*dnsSecurityToken = "fake-dns-token"
	*cdnSecurityToken = "fake-cdn-token"
	InitCLI()
	if Cfg.DNSCredentials.SecurityToken != *dnsSecurityToken || Cfg.CertCredentials.SecurityToken != *cdnSecurityToken {
		t.Fatal("CLI token configuration mismatch")
	}
	*dnsSecurityToken = ""
	*cdnSecurityToken = ""
	InitCLI()
	if Cfg.DNSCredentials.SecurityToken != "" || Cfg.CertCredentials.SecurityToken != "" {
		t.Fatal("legacy credentials inherited a token")
	}
}
func TestFactoriesKeepSTSCredentialsSeparate(t *testing.T) {
	var aopts applicant.Options
	var dopts deployer.Options
	applicant.RegisterApplicant("test-sts", func(opts ...applicant.Option) applicant.IApplicant { aopts.Update(opts...); return &fakeApplicant{} })
	deployer.RegisterDeployer("test-sts", func(opts ...deployer.Option) (deployer.IDeployer, error) {
		dopts.Update(opts...)
		return &fakeDeployer{}, nil
	})
	cfg := config.Config{DNSSiteID: "dns-site", CDNSiteID: "cdn-site", DNSRegion: "dns-region", CDNRegion: "cdn-region", DNSPlatform: "test-sts", CDNPlatform: "test-sts", DNSCredentials: config.ICredentials{SecurityToken: "fake-dns-token"}, CertCredentials: config.ICredentials{SecurityToken: "fake-cdn-token"}}
	if _, err := newApplicant(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := newDeployer(cfg); err != nil {
		t.Fatal(err)
	}
	if aopts.SiteID != cfg.DNSSiteID || dopts.SiteID != cfg.CDNSiteID || aopts.Region != cfg.DNSRegion || dopts.Region != cfg.CDNRegion {
		t.Fatal("site/region settings crossed between DNS and deployment")
	}
	if aopts.SecurityToken != cfg.DNSCredentials.SecurityToken || dopts.SecurityToken != cfg.CertCredentials.SecurityToken {
		t.Fatal("credentials crossed between DNS and certificate services")
	}
	cfg.DNSCredentials.SecurityToken = ""
	if _, err := newApplicant(cfg); err != nil {
		t.Fatal(err)
	}
	if aopts.SecurityToken != "" {
		t.Fatal("CDN token leaked to DNS")
	}
}
