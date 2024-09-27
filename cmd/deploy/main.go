package main

import (
	"fmt"
	"github.com/tiyee/AutoCert"
	"github.com/tiyee/AutoCert/internal/deployer"
)

func main() {
	cfg := AutoCert.Cfg
	AutoCert.ValidApplyCfg()
	opt := deployer.Option{
		Domain:          cfg.Domain,
		AccessKeyId:     cfg.CertCredentials.AccessKeyId,
		AccessKeySecret: cfg.CertCredentials.AccessKeySecret,
		Variables:       make(map[string]string),
	}
	aliyunCDN, err := deployer.NewAliyunCdn(&opt)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	lst, err := aliyunCDN.Search(opt.Domain)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for _, l := range lst {
		fmt.Println(l.CertId, l.CertName, l.Fingerprint, l.Renewable())
	}
}
