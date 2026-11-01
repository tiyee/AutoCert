package deployer

import (
	cdn "github.com/alibabacloud-go/cdn-20180510/v5/client"
	ssl "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"
	"testing"
)

func TestSTSClientCredentials(t *testing.T) {
	for _, token := range []string{"", "fake-cdn-sts-token"} {
		opts := []Option{WithAccessKeyId("fake-id"), WithAccessKeySecret("fake-secret"), WithSecurityToken(token)}
		d, err := NewAliyunCdn(opts...)
		if err != nil {
			t.Fatal(err)
		}
		client := d.(*AliyunCdn).client.(*cdn.Client)
		got, err := client.GetSecurityToken()
		if err != nil {
			t.Fatal(err)
		}
		if got == nil || *got != token {
			t.Fatal("Aliyun CDN token was not propagated")
		}
		tc, err := NewTencent(opts...)
		if err != nil {
			t.Fatal(err)
		}
		tcClient := tc.(*Tencent).client.(*ssl.Client)
		if tcClient.GetCredential().GetToken() != token {
			t.Fatal("Tencent SSL token was not propagated")
		}
	}
}
