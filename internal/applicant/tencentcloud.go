package applicant

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	dnspod "github.com/go-acme/tencentclouddnspod/v20210323"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

func init() { RegisterApplicant("tencent", NewTencentYun) }

type tencentYun struct{ option *Options }

func NewTencentYun(option ...Option) IApplicant {
	opts := &Options{}
	opts.Update(option...)
	return &tencentYun{option: opts}
}
func (a *tencentYun) Apply() (*Certificate, error) {
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
			req := dnspod.NewCreateRecordRequest()
			req.Domain = common.StringPtr(zone)
			req.SubDomain = common.StringPtr(rr)
			req.RecordType = common.StringPtr("TXT")
			req.RecordLine = common.StringPtr("默认")
			req.Value = common.StringPtr(value)
			req.TTL = common.Uint64Ptr(600)
			rsp, err := dnspod.CreateRecordWithContext(context.Background(), client, req)
			if err != nil {
				return recordHandle{}, tencentDNSError("create", err)
			}
			if rsp == nil || rsp.Response == nil || rsp.Response.RecordId == nil || *rsp.Response.RecordId == 0 {
				return recordHandle{}, fmt.Errorf("Tencent DNS did not return a record ID")
			}
			return recordHandle{zone: zone, id: strconv.FormatUint(*rsp.Response.RecordId, 10)}, nil
		},
		remove: func(record recordHandle) error {
			id, err := strconv.ParseUint(record.id, 10, 64)
			if err != nil {
				return err
			}
			req := dnspod.NewDeleteRecordRequest()
			req.Domain = common.StringPtr(record.zone)
			req.RecordId = common.Uint64Ptr(id)
			rsp, err := dnspod.DeleteRecordWithContext(context.Background(), client, req)
			var sdkErr *sdkerrors.TencentCloudSDKError
			if errors.As(err, &sdkErr) && sdkErr.Code == "ResourceNotFound.NoDataOfRecord" {
				return nil
			}
			if err == nil && (rsp == nil || rsp.Response == nil) {
				return fmt.Errorf("Tencent DNS delete returned an empty response")
			}
			return tencentDNSError("delete", err)
		},
	}
	return apply(a.option, provider)
}
func tencentDNSError(operation string, err error) error {
	if err == nil {
		return nil
	}
	var sdkErr *sdkerrors.TencentCloudSDKError
	if errors.As(err, &sdkErr) {
		return fmt.Errorf("Tencent DNS %s failed (%s)", operation, sdkErr.Code)
	}
	return fmt.Errorf("Tencent DNS %s failed (%T)", operation, err)
}

func (a *tencentYun) createClient() (*dnspod.Client, error) {
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "dnspod.tencentcloudapi.com"
	cpf.HttpProfile.ReqTimeout = 100
	return dnspod.NewClient(common.NewTokenCredential(a.option.AccessKeyId, a.option.AccessKeySecret, a.option.SecurityToken), "ap-beijing", cpf)
}
