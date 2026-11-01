package AutoCert

import (
	"flag"
	"os"
	"testing"

	"github.com/tiyee/AutoCert/internal/config"
)

func TestInitCLIReturnsWithoutExiting(t *testing.T) {
	oldArgs, oldCfg := os.Args, Cfg
	oldDomain, oldEmail, oldVersion, oldRenew := *domain, *email, *version, *renew
	oldCertOnly := *certOnly
	oldName, oldHandling := flag.CommandLine.Name(), flag.CommandLine.ErrorHandling()
	t.Cleanup(func() {
		os.Args, Cfg = oldArgs, oldCfg
		*domain, *email, *version, *renew = oldDomain, oldEmail, oldVersion, oldRenew
		*certOnly = oldCertOnly
		flag.CommandLine.Init(oldName, oldHandling)
	})

	for _, tc := range []struct {
		name string
		args []string
		want bool
	}{
		{name: "missing domain and email"},
		{name: "missing email", args: []string{"--domain=example.invalid"}},
		{name: "missing domain", args: []string{"--email=test@example.invalid"}},
		{name: "help", args: []string{"--help"}},
		{name: "version", args: []string{"--version"}},
		{name: "unknown flag", args: []string{"--unknown-option"}},
		{name: "invalid flag value", args: []string{"--certonly=invalid"}},
		{name: "valid", args: []string{"--domain=example.invalid", "--email=test@example.invalid"}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			*domain, *email, *version = "", "", false
			*certOnly = false
			os.Args = append([]string{"autocert"}, tc.args...)
			if got := InitCLI(); got != tc.want {
				t.Fatalf("InitCLI() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCredentialValidationReturnsWithoutExiting(t *testing.T) {
	oldCfg := Cfg
	t.Cleanup(func() { Cfg = oldCfg })
	for _, tc := range []struct {
		name        string
		credentials config.ICredentials
		want        bool
	}{
		{name: "missing pair"},
		{name: "missing secret", credentials: config.ICredentials{AccessKeyId: "mock-id"}},
		{name: "missing id", credentials: config.ICredentials{AccessKeySecret: "mock-secret"}},
		{name: "complete pair", credentials: config.ICredentials{AccessKeyId: "mock-id", AccessKeySecret: "mock-secret"}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			Cfg = config.Config{Domain: "example.invalid", DNSCredentials: tc.credentials, CertCredentials: tc.credentials}
			if got := ValidApplyCfg(); got != tc.want {
				t.Fatalf("ValidApplyCfg() = %v, want %v", got, tc.want)
			}
			if got := ValidDeployCfg(); got != tc.want {
				t.Fatalf("ValidDeployCfg() = %v, want %v", got, tc.want)
			}
		})
	}
}
