package main

import (
	"fmt"
	"os"

	"github.com/tiyee/AutoCert"
)

func main() {
	if !AutoCert.InitCLI() || !AutoCert.ValidApplyCfg() || !AutoCert.ValidDeployCfg() {
		return
	}
	var err error
	if AutoCert.Cfg.CertOnly {
		err = AutoCert.CertOnly(AutoCert.Cfg)
	} else {
		err = AutoCert.Renew(AutoCert.Cfg)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] %v\n", AutoCert.Cfg.Domain, err)
	}
}
