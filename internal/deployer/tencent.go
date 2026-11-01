package deployer

import (
	"context"
	"fmt"
	"strings"
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

type tencentAPI interface {
	UploadCertificateWithContext(context.Context, *ssl.UploadCertificateRequest) (*ssl.UploadCertificateResponse, error)
	DescribeCertificates(*ssl.DescribeCertificatesRequest) (*ssl.DescribeCertificatesResponse, error)
}

type Tencent struct {
	client  tencentAPI
	options *Options
}

func NewTencent(option ...Option) (IDeployer, error) {
	opts := &Options{}
	opts.Update(option...)
	c, err := ssl.NewClient(common.NewTokenCredential(opts.AccessKeyId, opts.AccessKeySecret, opts.SecurityToken), "ap-guangzhou", profile.NewClientProfile())
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

	rsp, err := t.client.UploadCertificateWithContext(ctx, req)
	if err != nil {
		return err
	}
	if rsp == nil || rsp.Response == nil || tea.StringValue(rsp.Response.CertificateId) == "" {
		return fmt.Errorf("upload returned no certificate ID")
	}
	return nil
}

func (t *Tencent) Search(domain string) ([]CertStat, error) {
	req := ssl.NewDescribeCertificatesRequest()
	// 搜索关键词不保证匹配泛域名或 SAN，完整分页后按覆盖关系过滤。
	req.Limit = tea.Uint64(100)
	req.ExpirationSort = tea.String("DESC")
	req.CertificateType = tea.String("SVR")
	var results []CertStat
	for offset := uint64(0); ; {
		req.Offset = tea.Uint64(offset)
		rsp, err := t.client.DescribeCertificates(req)
		if err != nil {
			return nil, err
		}
		if rsp == nil || rsp.Response == nil || rsp.Response.TotalCount == nil {
			return nil, fmt.Errorf("query certificates returned an invalid response")
		}
		for _, v := range rsp.Response.Certificates {
			if v == nil {
				return nil, fmt.Errorf("query certificates returned an empty item")
			}
			if v.Status == nil || (*v.Status != 1 && *v.Status != 3) {
				continue
			}
			names := append([]*string{v.Domain}, v.SubjectAltName...)
			matches := false
			for _, name := range names {
				if coversDomain(tea.StringValue(name), domain) {
					matches = true
					break
				}
			}
			if !matches {
				continue
			}
			end, err := parseTencentExpiry(tea.StringValue(v.CertEndTime))
			if err != nil {
				return nil, err
			}
			results = append(results, CertStat{LastTime: end.Unix(), CertName: tea.StringValue(v.Alias), CertId: tea.StringValue(v.CertificateId)})
		}
		offset += uint64(len(rsp.Response.Certificates))
		if offset >= *rsp.Response.TotalCount {
			break
		}
		if len(rsp.Response.Certificates) == 0 {
			return nil, fmt.Errorf("query certificates returned an incomplete page")
		}
	}
	return results, nil
}
func parseTencentExpiry(value string) (time.Time, error) {
	end, err := time.ParseInLocation(time.DateTime, value, time.FixedZone("UTC+8", 8*3600))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid Tencent certificate expiration")
	}
	return end, nil
}
func coversDomain(pattern, domain string) bool {
	pattern = strings.ToLower(strings.TrimSuffix(pattern, "."))
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	if pattern == "" || domain == "" {
		return false
	}
	if pattern == domain {
		return true
	}
	if strings.HasPrefix(domain, "*.") || !strings.HasPrefix(pattern, "*.") {
		return false
	}
	suffix := strings.TrimPrefix(pattern, "*")
	return strings.HasSuffix(domain, suffix) && !strings.Contains(strings.TrimSuffix(domain, suffix), ".") && len(domain) > len(suffix)
}

func (t *Tencent) SetCertificate(cert applicant.Certificate) {
	t.options.Certificate = cert
}
