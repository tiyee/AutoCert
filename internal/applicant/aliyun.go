package applicant

import (
	"github.com/go-acme/lego/v4/providers/dns/alidns"
	"time"
)

type aliyun struct {
	option *ApplyOption
}

func NewAliyun(option *ApplyOption) IApplicant {
	return &aliyun{
		option: option,
	}
}

func (a *aliyun) Apply() (*Certificate, error) {
	cfg := alidns.Config{
		APIKey:             a.option.AccessKeyId,
		SecretKey:          a.option.AccessKeySecret,
		RegionID:           "cn-hangzhou",
		TTL:                600,
		HTTPTimeout:        100 * time.Second,
		PropagationTimeout: 200 * time.Second,
		PollingInterval:    5 * time.Second,
	}
	dnsProvider, err := alidns.NewDNSProviderConfig(&cfg)
	if err != nil {
		return nil, err
	}

	return apply(a.option, dnsProvider)
}
