package AutoCert

import (
	"context"
	"fmt"

	"github.com/tiyee/AutoCert/internal/applicant"
	"github.com/tiyee/AutoCert/internal/config"
	"github.com/tiyee/AutoCert/internal/deployer"
)

func CertOnly(cfg config.Config) {
	af := applicant.GetApplicant(cfg.Platform)
	if af == nil {
		fmt.Println("apply Platform not supported")
		return
	}

	applyClient := af(applicant.WithDomain(cfg.Domain),
		applicant.WithEmail(cfg.Email),
		applicant.WithAccessKeyId(cfg.DNSCredentials.AccessKeyId),
		applicant.WithAccessKeySecret(cfg.DNSCredentials.AccessKeySecret),
		applicant.WithNameservers("1.1.1.1;8.8.8.8"))
	certificate, err := applyClient.Apply()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	df := deployer.GetDeployer(cfg.Platform)
	if df == nil {
		fmt.Println("deploy Platform not supported")
		return
	}
	sslClient, err := df(deployer.WithDomain(cfg.Domain),
		deployer.WithAccessKeyId(cfg.CertCredentials.AccessKeyId),
		deployer.WithAccessKeySecret(cfg.CertCredentials.AccessKeySecret),
		deployer.WithCertificate(*certificate),
		deployer.WithVariables(make(map[string]string)))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if err = sslClient.Deploy(context.Background()); err == nil {
		fmt.Println("Successfully deployed")
	} else {
		fmt.Println(err.Error())
	}
}
