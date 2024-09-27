package AutoCert

import (
	"context"
	"fmt"
	"github.com/tiyee/AutoCert/internal/applicant"
	"github.com/tiyee/AutoCert/internal/config"
	"github.com/tiyee/AutoCert/internal/deployer"
)

func Renew(cfg config.Config) {
	cdnOpt := deployer.Option{
		Domain:          cfg.Domain,
		AccessKeyId:     cfg.CertCredentials.AccessKeyId,
		AccessKeySecret: cfg.CertCredentials.AccessKeySecret,
		Variables:       make(map[string]string),
	}
	aliyunCDN, err := deployer.NewAliyunCdn(&cdnOpt)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	certs, err := aliyunCDN.Search(cfg.Domain)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if len(certs) == 0 {
		fmt.Println("no certs found")
		return
	}
	if !certs[0].Renewable() {
		fmt.Println("Certificate no need renew")
		return
	}
	applyOpt := applicant.ApplyOption{
		Email:           cfg.Email,
		Domain:          cfg.Domain,
		AccessKeyId:     cfg.DNSCredentials.AccessKeyId,
		AccessKeySecret: cfg.DNSCredentials.AccessKeySecret,
		Nameservers:     "1.1.1.1;8.8.8.8",
	}
	aliyunDNS := applicant.NewAliyun(&applyOpt)
	certificate, err := aliyunDNS.Apply()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if err = aliyunCDN.WithOptions(func(opt *deployer.Option) {
		opt.Certificate = *certificate
	}).Deploy(context.Background()); err == nil {
		fmt.Println("Successfully deployed")
	} else {
		fmt.Println(err.Error())
	}
}
