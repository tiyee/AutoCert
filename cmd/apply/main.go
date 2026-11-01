package main

import (
	"fmt"
	"os"

	"github.com/tiyee/AutoCert"
	"github.com/tiyee/AutoCert/internal/applicant"
)

func main() {
	if !AutoCert.InitCLI() || !AutoCert.ValidApplyCfg() {
		return
	}
	cfg := AutoCert.Cfg
	f := applicant.GetApplicant(cfg.DNSPlatform)
	if f == nil {
		fmt.Fprintf(os.Stderr, "[%s] apply platform %q not supported\n", cfg.Domain, cfg.DNSPlatform)
		return
	}
	client := f(applicant.WithDomain(cfg.Domain),
		applicant.WithEmail(cfg.Email),
		applicant.WithAccessKeyId(cfg.DNSCredentials.AccessKeyId),
		applicant.WithAccessKeySecret(cfg.DNSCredentials.AccessKeySecret), applicant.WithSecurityToken(cfg.DNSCredentials.SecurityToken), applicant.WithSiteID(cfg.DNSSiteID), applicant.WithRegion(cfg.DNSRegion),
		applicant.WithNameservers("1.1.1.1;8.8.8.8"))
	lst, err := client.Apply()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] apply: %v\n", cfg.Domain, err)
		return
	}
	fmt.Println(lst)
}
