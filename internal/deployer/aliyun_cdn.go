package deployer

import (
	"context"
	"fmt"
	cdn20180510 "github.com/alibabacloud-go/cdn-20180510/v5/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"net/http"
	"sort"
	"strconv"
	"time"
)

type AliyunCdn struct {
	client *cdn20180510.Client
	option *Option
}

func (a *AliyunCdn) Deploy(ctx context.Context) error {
	certName := fmt.Sprintf("%s-%s", a.option.Domain, time.Now().Format(time.DateTime))
	setCdnDomainSSLCertificateRequest := &cdn20180510.SetCdnDomainSSLCertificateRequest{
		DomainName:  tea.String(a.option.Domain),
		CertName:    tea.String(certName),
		CertType:    tea.String("upload"),
		SSLProtocol: tea.String("on"),
		SSLPub:      tea.String(a.option.Certificate.Certificate),
		SSLPri:      tea.String(a.option.Certificate.PrivateKey),
		CertRegion:  tea.String("cn-hangzhou"),
	}

	runtime := &util.RuntimeOptions{}

	resp, err := a.client.SetCdnDomainSSLCertificateWithOptions(setCdnDomainSSLCertificateRequest, runtime)
	if err != nil {
		return err
	}
	if tea.Int32Value(resp.StatusCode) < 200 || tea.Int32Value(resp.StatusCode) >= 300 {
		return fmt.Errorf("set cdn domain ssl certificate failed")
	}

	return nil
}

func (a *AliyunCdn) Search(domain string) ([]CertStat, error) {
	describeCdnSSLCertificateListRequest := &cdn20180510.DescribeCdnSSLCertificateListRequest{
		DomainName:    tea.String(a.option.Domain),
		PageNumber:    tea.Int64(1),
		PageSize:      tea.Int64(20),
		SearchKeyword: tea.String(domain),
	}
	runtime := &util.RuntimeOptions{}
	resp, err := a.client.DescribeCdnSSLCertificateListWithOptions(describeCdnSSLCertificateListRequest, runtime)
	if err != nil {
		return nil, err
	}
	if tea.Int32Value(resp.StatusCode) != http.StatusOK {
		return nil, err
	}
	total := tea.Int32Value(resp.Body.CertificateListModel.Count)
	if total < 1 {
		return []CertStat{}, err
	}
	results := make([]CertStat, 0, total)
	for _, item := range resp.Body.CertificateListModel.CertList.Cert {
		results = append(results, CertStat{
			LastTime:    tea.Int64Value(item.LastTime) / 1000,
			Fingerprint: tea.StringValue(item.Fingerprint),
			CertName:    tea.StringValue(item.CertName),
			CertId:      strconv.FormatInt(tea.Int64Value(item.CertId), 10),
		})
	}
	sort.Sort(sort.Reverse(CertStats(results)))
	return results, nil
}

func NewAliyunCdn(option *Option) (*AliyunCdn, error) {
	a := &AliyunCdn{
		option: option,
	}
	client, err := a.createClient(option.AccessKeyId, option.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	return &AliyunCdn{
		client: client,
		option: option,
	}, nil
}
func (a *AliyunCdn) createClient(accessKeyId, accessKeySecret string) (_result *cdn20180510.Client, _err error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
	}
	config.Endpoint = tea.String("cdn.aliyuncs.com")
	_result = &cdn20180510.Client{}
	_result, _err = cdn20180510.NewClient(config)
	return _result, _err
}
func (a *AliyunCdn) WithOptions(fn func(opt *Option)) *AliyunCdn {
	fn(a.option)
	return a

}
