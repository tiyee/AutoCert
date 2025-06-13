package main

import (
	"fmt"

	"github.com/tiyee/AutoCert"
	"github.com/tiyee/AutoCert/internal/applicant"
)

func main() {
	cfg := AutoCert.Cfg
	AutoCert.ValidApplyCfg()
	f := applicant.GetApplicant(cfg.Platform)
	if f == nil {
		fmt.Println("apply Platform not supported")
		return
	}
	client := f(applicant.WithDomain(cfg.Domain),
		applicant.WithEmail(cfg.Email),
		applicant.WithAccessKeyId(cfg.DNSCredentials.AccessKeyId),
		applicant.WithAccessKeySecret(cfg.DNSCredentials.AccessKeySecret),
		applicant.WithNameservers("1.1.1.1;8.8.8.8"))
	lst, err := client.Apply()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(lst)
}
