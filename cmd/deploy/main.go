package main

import (
	"fmt"

	"github.com/tiyee/AutoCert"
	"github.com/tiyee/AutoCert/internal/deployer"
)

func main() {
	cfg := AutoCert.Cfg
	AutoCert.ValidApplyCfg()
	df := deployer.GetDeployer(cfg.Platform)
	if df == nil {
		fmt.Println("deploy Platform not supported")
		return
	}
	sslClient, err := df(deployer.WithDomain(cfg.Domain),
		deployer.WithAccessKeyId(cfg.CertCredentials.AccessKeyId),
		deployer.WithAccessKeySecret(cfg.CertCredentials.AccessKeySecret),
		deployer.WithVariables(make(map[string]string)))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	lst, err := sslClient.Search(cfg.Domain)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for _, l := range lst {
		fmt.Println(l.CertId, l.CertName, l.Fingerprint, l.Renewable())
	}
}
