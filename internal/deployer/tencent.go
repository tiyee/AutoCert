package deployer

import (
	"context"
	"fmt"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ssl "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"

	"github.com/tiyee/AutoCert/internal/applicant"
)

func init() {
	RegisterDeployer("tencent", NewTencent)
}

type Tencent struct {
	client  *ssl.Client
	options *Options
}

func NewTencent(option ...Option) (IDeployer, error) {
	opts := &Options{}
	opts.Update(option...)
	c, err := ssl.NewClient(common.NewCredential(opts.AccessKeyId, opts.AccessKeySecret), "ap-guangzhou", profile.NewClientProfile())
	if err != nil {
		return nil, err
	}
	return &Tencent{
		client:  c,
		options: opts,
	}, nil
}

func (t *Tencent) Deploy(ctx context.Context) error {
	req := ssl.NewUploadCertificateRequest()
	req.CertificatePublicKey = tea.String(t.options.Certificate.Certificate)
	req.CertificatePrivateKey = tea.String(t.options.Certificate.PrivateKey)

	rsp, err := t.client.UploadCertificate(req)
	if err != nil {
		return err
	}
	fmt.Println(rsp.ToJsonString())
	return nil
}

func (t *Tencent) Search(domain string) ([]CertStat, error) {
	req := ssl.NewDescribeCertificatesRequest()
	req.SearchKey = tea.String(domain)
	rsp, err := t.client.DescribeCertificates(req)
	if err != nil {
		return nil, err
	}
	total := tea.Uint64Value(rsp.Response.TotalCount)
	if total < 1 {
		return []CertStat{}, err
	}
	results := make([]CertStat, 0, total)
	for _, v := range rsp.Response.Certificates {
		end, err := time.Parse("2006-01-02 15:04:05", *v.CertEndTime)
		if err != nil {
			end = time.Now()
			fmt.Println(err)
		}
		results = append(results, CertStat{
			LastTime: end.Unix(),
			CertName: tea.StringValue(v.Alias),
			CertId:   tea.StringValue(v.CertificateId),
		})
	}
	return results, nil
}

func (t *Tencent) SetCertificate(cert applicant.Certificate) {
	t.options.Certificate = cert
}
