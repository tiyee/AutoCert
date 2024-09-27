package main

import (
	"fmt"
	"github.com/tiyee/AutoCert"
	"github.com/tiyee/AutoCert/internal/applicant"
)

func main() {
	cfg := AutoCert.Cfg
	AutoCert.ValidApplyCfg()
	opt := applicant.ApplyOption{
		Email:           cfg.Email,
		Domain:          cfg.Domain,
		AccessKeyId:     cfg.DNSCredentials.AccessKeyId,
		AccessKeySecret: cfg.DNSCredentials.AccessKeySecret,
		Nameservers:     "1.1.1.1;8.8.8.8",
	}
	aliyunDNS := applicant.NewAliyun(&opt)
	lst, err := aliyunDNS.Apply()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(lst)
}
