package main

import (
	"fmt"
	"os"

	"github.com/tiyee/AutoCert"
	"github.com/tiyee/AutoCert/internal/deployer"
)

func main() {
	AutoCert.InitCLI()
	cfg := AutoCert.Cfg
	AutoCert.ValidDeployCfg()
	df := deployer.GetDeployer(cfg.CDNPlatform)
	if df == nil {
		fmt.Fprintln(os.Stderr, "deploy Platform not supported")
		os.Exit(1)
	}
	sslClient, err := df(deployer.WithDomain(cfg.Domain),
		deployer.WithAccessKeyId(cfg.CertCredentials.AccessKeyId),
		deployer.WithAccessKeySecret(cfg.CertCredentials.AccessKeySecret), deployer.WithSecurityToken(cfg.CertCredentials.SecurityToken), deployer.WithSiteID(cfg.CDNSiteID), deployer.WithRegion(cfg.CDNRegion),
		deployer.WithVariables(make(map[string]string)))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lst, err := sslClient.Search(cfg.Domain)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, l := range lst {
		fmt.Println(l.CertId, l.CertName, l.Fingerprint, l.Renewable())
	}
}
