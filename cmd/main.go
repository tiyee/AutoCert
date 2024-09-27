package main

import "github.com/tiyee/AutoCert"

func main() {
	AutoCert.ValidApplyCfg()
	AutoCert.ValidDeployCfg()
	if AutoCert.Cfg.CertOnly {
		AutoCert.CertOnly(AutoCert.Cfg)
	} else {
		AutoCert.Renew(AutoCert.Cfg)
	}

}
