package main

import (
	"fmt"
	"os"

	"github.com/tiyee/AutoCert"
	"github.com/tiyee/AutoCert/internal/applicant"
)

func main() {
	AutoCert.InitCLI()
	cfg := AutoCert.Cfg
	AutoCert.ValidApplyCfg()
	f := applicant.GetApplicant(cfg.DNSPlatform)
	if f == nil {
		fmt.Fprintln(os.Stderr, "apply Platform not supported")
		os.Exit(1)
	}
	client := f(applicant.WithDomain(cfg.Domain),
		applicant.WithEmail(cfg.Email),
		applicant.WithAccessKeyId(cfg.DNSCredentials.AccessKeyId),
		applicant.WithAccessKeySecret(cfg.DNSCredentials.AccessKeySecret), applicant.WithSecurityToken(cfg.DNSCredentials.SecurityToken), applicant.WithSiteID(cfg.DNSSiteID), applicant.WithRegion(cfg.DNSRegion),
		applicant.WithNameservers("1.1.1.1;8.8.8.8"))
	lst, err := client.Apply()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(lst)
}
