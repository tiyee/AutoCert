package main

import (
	"fmt"
	"os"

	"github.com/tiyee/AutoCert"
	"github.com/tiyee/AutoCert/internal/deployer"
)

func main() {
	if !AutoCert.InitCLI() || !AutoCert.ValidDeployCfg() {
		return
	}
	cfg := AutoCert.Cfg
	df := deployer.GetDeployer(cfg.CDNPlatform)
	if df == nil {
		fmt.Fprintf(os.Stderr, "[%s] deploy platform %q not supported\n", cfg.Domain, cfg.CDNPlatform)
		return
	}
	sslClient, err := df(deployer.WithDomain(cfg.Domain),
		deployer.WithAccessKeyId(cfg.CertCredentials.AccessKeyId),
		deployer.WithAccessKeySecret(cfg.CertCredentials.AccessKeySecret), deployer.WithSecurityToken(cfg.CertCredentials.SecurityToken), deployer.WithSiteID(cfg.CDNSiteID), deployer.WithRegion(cfg.CDNRegion),
		deployer.WithVariables(make(map[string]string)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] deploy: %v\n", cfg.Domain, err)
		return
	}
	lst, err := sslClient.Search(cfg.Domain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] search: %v\n", cfg.Domain, err)
		return
	}
	for _, l := range lst {
		fmt.Println(l.CertId, l.CertName, l.Fingerprint, l.Renewable())
	}
}
