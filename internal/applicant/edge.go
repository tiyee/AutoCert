package applicant

import (
	"context"
	"time"

	"github.com/tiyee/AutoCert/internal/edge"
)

func init() { RegisterApplicant("esa", NewESA); RegisterApplicant("teo", NewTEO) }

type edgeApplicant struct {
	options  *Options
	platform string
}

func NewESA(opts ...Option) IApplicant {
	o := &Options{}
	o.Update(opts...)
	return &edgeApplicant{options: o, platform: "esa"}
}
func NewTEO(opts ...Option) IApplicant {
	o := &Options{}
	o.Update(opts...)
	return &edgeApplicant{options: o, platform: "teo"}
}

type edgeDNS interface {
	CreateTXT(context.Context, string, string) (edge.Record, error)
	DeleteTXT(context.Context, edge.Record) error
}

// EdgeOne 的 DNS 传播可能更慢，使用其 lego 适配器相同的 20 分钟上限。
type edgeDNSProvider struct{ *ownedDNSProvider }

func (p *edgeDNSProvider) Timeout() (time.Duration, time.Duration) {
	return 20 * time.Minute, 5 * time.Second
}
func edgeProvider(client edgeDNS) *edgeDNSProvider {
	return &edgeDNSProvider{&ownedDNSProvider{
		create: func(fqdn, value string) (recordHandle, error) {
			record, err := client.CreateTXT(context.Background(), fqdn, value)
			return recordHandle{zone: record.SiteID, id: record.ID}, err
		},
		remove: func(record recordHandle) error {
			return client.DeleteTXT(context.Background(), edge.Record{SiteID: record.zone, ID: record.id})
		},
	}}
}
func (a *edgeApplicant) Apply() (*Certificate, error) {
	o := a.options
	var client edgeDNS
	var err error
	if a.platform == "esa" {
		client, err = edge.NewESA(o.AccessKeyId, o.AccessKeySecret, o.SecurityToken, o.Region, o.SiteID)
	} else {
		client, err = edge.NewTEO(o.AccessKeyId, o.AccessKeySecret, o.SecurityToken, o.Region, o.SiteID)
	}
	if err != nil {
		return nil, err
	}
	return apply(o, edgeProvider(client))
}
