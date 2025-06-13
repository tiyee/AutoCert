package applicant

import (
	"time"

	"github.com/go-acme/lego/v4/providers/dns/tencentcloud"
)

func init() {
	RegisterApplicant("tencent", NewTencentYun)
}

type tencentYun struct {
	option *Options
}

func NewTencentYun(option ...Option) IApplicant {
	opts := &Options{}
	opts.Update(option...)
	return &tencentYun{
		option: opts,
	}
}

func (a *tencentYun) Apply() (*Certificate, error) {
	cfg := tencentcloud.Config{
		SecretID:           a.option.AccessKeyId,
		SecretKey:          a.option.AccessKeySecret,
		Region:             "ap-beijing",
		PropagationTimeout: 200 * time.Second,
		PollingInterval:    5 * time.Second,
		TTL:                600,
		HTTPTimeout:        100 * time.Second,
	}
	dnsProvider, err := tencentcloud.NewDNSProviderConfig(&cfg)
	if err != nil {
		return nil, err
	}

	return apply(a.option, dnsProvider)
}
