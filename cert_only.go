package AutoCert

import (
	"context"
	"fmt"
	"github.com/tiyee/AutoCert/internal/applicant"
	"github.com/tiyee/AutoCert/internal/config"
	"github.com/tiyee/AutoCert/internal/deployer"
)

func CertOnly(cfg config.Config) {
	applyOpt := applicant.ApplyOption{
		Email:           cfg.Email,
		Domain:          cfg.Domain,
		AccessKeyId:     cfg.DNSCredentials.AccessKeyId,
		AccessKeySecret: cfg.DNSCredentials.AccessKeySecret,
		Nameservers:     "1.1.1.1;8.8.8.8",
	}
	fmt.Println(applyOpt)
	aliyunDNS := applicant.NewAliyun(&applyOpt)
	certificate, err := aliyunDNS.Apply()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	cdnOpt := deployer.Option{
		Domain:          cfg.Domain,
		AccessKeyId:     cfg.CertCredentials.AccessKeyId,
		AccessKeySecret: cfg.CertCredentials.AccessKeySecret,
		Certificate:     *certificate,
		Variables:       make(map[string]string),
	}
	aliyunCDN, err := deployer.NewAliyunCdn(&cdnOpt)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if err = aliyunCDN.Deploy(context.Background()); err == nil {
		fmt.Println("Successfully deployed")
	} else {
		fmt.Println(err.Error())
	}
}
