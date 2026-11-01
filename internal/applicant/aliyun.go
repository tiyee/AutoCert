package applicant

import (
	"context"
	"errors"
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
	alidns "github.com/go-acme/alidns-20150109/v4/client"
)

func init() { RegisterApplicant("aliyun", NewAliyun) }

type aliyun struct{ options *Options }

func NewAliyun(option ...Option) IApplicant {
	opts := &Options{}
	opts.Update(option...)
	return &aliyun{options: opts}
}
func (a *aliyun) Apply() (*Certificate, error) {
	client, err := a.createClient()
	if err != nil {
		return nil, err
	}
	provider := &ownedDNSProvider{
		create: func(fqdn, value string) (recordHandle, error) {
			zone, rr, err := recordLocation(fqdn)
			if err != nil {
				return recordHandle{}, err
			}
			rsp, err := alidns.AddDomainRecordWithContext(context.Background(), client, &alidns.AddDomainRecordRequest{DomainName: tea.String(zone), RR: tea.String(rr), Type: tea.String("TXT"), Value: tea.String(value), TTL: tea.Int64(600)}, &dara.RuntimeOptions{Autoretry: tea.Bool(false)})
			if err != nil {
				return recordHandle{}, dnsAPIError("create", err)
			}
			if rsp == nil || rsp.Body == nil {
				return recordHandle{}, fmt.Errorf("Aliyun DNS returned an empty response")
			}
			return recordHandle{zone: zone, id: tea.StringValue(rsp.Body.RecordId)}, nil
		},
		remove: func(record recordHandle) error {
			rsp, err := alidns.DeleteDomainRecordWithContext(context.Background(), client, &alidns.DeleteDomainRecordRequest{RecordId: tea.String(record.id)}, &dara.RuntimeOptions{Autoretry: tea.Bool(false)})
			var sdkErr *tea.SDKError
			if errors.As(err, &sdkErr) && tea.StringValue(sdkErr.Code) == "InvalidRecordId.NotFound" {
				return nil
			}
			if err == nil && (rsp == nil || rsp.Body == nil) {
				return fmt.Errorf("Aliyun DNS delete returned an empty response")
			}
			return dnsAPIError("delete", err)
		},
	}
	return apply(a.options, provider)
}

// SDK 错误文本可能包含敏感请求信息，仅输出错误码。
func dnsAPIError(operation string, err error) error {
	if err == nil {
		return nil
	}
	var sdkErr *tea.SDKError
	if errors.As(err, &sdkErr) {
		return fmt.Errorf("Aliyun DNS %s failed (%s)", operation, tea.StringValue(sdkErr.Code))
	}
	return fmt.Errorf("Aliyun DNS %s failed (%T)", operation, err)
}

func (a *aliyun) createClient() (*alidns.Client, error) {
	return alidns.NewClient(&openapi.Config{AccessKeyId: tea.String(a.options.AccessKeyId), AccessKeySecret: tea.String(a.options.AccessKeySecret), SecurityToken: tea.String(a.options.SecurityToken), Endpoint: tea.String("alidns.aliyuncs.com"), ConnectTimeout: tea.Int(10000), ReadTimeout: tea.Int(100000)})
}
