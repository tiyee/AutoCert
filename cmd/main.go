package main

import (
	"fmt"
	"os"

	"github.com/tiyee/AutoCert"
)

func main() {
	AutoCert.InitCLI()
	AutoCert.ValidApplyCfg()
	AutoCert.ValidDeployCfg()
	var err error
	if AutoCert.Cfg.CertOnly {
		err = AutoCert.CertOnly(AutoCert.Cfg)
	} else {
		err = AutoCert.Renew(AutoCert.Cfg)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
