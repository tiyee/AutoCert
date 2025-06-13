package applicant

import (
	"time"

	"github.com/go-acme/lego/v4/providers/dns/alidns"
)

func init() {
	RegisterApplicant("aliyun", NewAliyun)
}

type aliyun struct {
	options *Options
}

func NewAliyun(option ...Option) IApplicant {
	opts := &Options{}
	opts.Update(option...)
	return &aliyun{
		options: opts,
	}
}

func (a *aliyun) Apply() (*Certificate, error) {
	cfg := alidns.Config{
		APIKey:             a.options.AccessKeyId,
		SecretKey:          a.options.AccessKeySecret,
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

	return apply(a.options, dnsProvider)
}
