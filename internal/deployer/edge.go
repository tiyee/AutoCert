package deployer

import (
	"context"

	"github.com/tiyee/AutoCert/internal/applicant"
	"github.com/tiyee/AutoCert/internal/edge"
)

func init() { RegisterDeployer("esa", NewESA); RegisterDeployer("teo", NewTEO) }

type edgeAPI interface {
	CheckTarget(context.Context, string) error
	Deploy(context.Context, string, string, string) error
	Certificates(context.Context, string) ([]edge.Certificate, error)
}
type edgeDeployer struct {
	client            edgeAPI
	options           *Options
	boundCertificates bool
}

func NewESA(opts ...Option) (IDeployer, error) {
	o := &Options{}
	o.Update(opts...)
	c, err := edge.NewESA(o.AccessKeyId, o.AccessKeySecret, o.SecurityToken, o.Region, o.SiteID)
	if err != nil {
		return nil, err
	}
	return &edgeDeployer{client: c, options: o}, nil
}
func NewTEO(opts ...Option) (IDeployer, error) {
	o := &Options{}
	o.Update(opts...)
	c, err := edge.NewTEO(o.AccessKeyId, o.AccessKeySecret, o.SecurityToken, o.Region, o.SiteID)
	if err != nil {
		return nil, err
	}
	return &edgeDeployer{client: c, options: o, boundCertificates: true}, nil
}
func (d *edgeDeployer) SetCertificate(cert applicant.Certificate) { d.options.Certificate = cert }
func (d *edgeDeployer) Deploy(ctx context.Context) error {
	return d.client.Deploy(ctx, d.options.Domain, d.options.Certificate.Certificate, d.options.Certificate.PrivateKey)
}
func (d *edgeDeployer) Search(domain string) ([]CertStat, error) {
	certs, err := d.client.Certificates(context.Background(), domain)
	if err != nil {
		return nil, err
	}
	results := make([]CertStat, 0, len(certs))
	for _, c := range certs {
		results = append(results, CertStat{CertId: c.ID, CertName: c.Name, Fingerprint: c.Fingerprint, LastTime: c.Expires.Unix()})
	}
	return results, nil
}

func (d *edgeDeployer) Preflight(ctx context.Context) error {
	return d.client.CheckTarget(ctx, d.options.Domain)
}
func (d *edgeDeployer) SelectRenewalCertificate(certs []CertStat) (CertStat, bool) {
	if !d.boundCertificates {
		return LatestCertificate(certs)
	}
	if len(certs) == 0 {
		return CertStat{}, false
	}
	earliest := certs[0]
	for _, cert := range certs[1:] {
		if cert.LastTime < earliest.LastTime {
			earliest = cert
		}
	}
	return earliest, true
}
