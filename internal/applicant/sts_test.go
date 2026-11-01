package applicant

import "testing"

func TestSTSClientCredentials(t *testing.T) {
	for _, token := range []string{"", "fake-dns-sts-token"} {
		opts := []Option{WithAccessKeyId("fake-id"), WithAccessKeySecret("fake-secret"), WithSecurityToken(token)}
		ali := NewAliyun(opts...).(*aliyun)
		client, err := ali.createClient()
		if err != nil {
			t.Fatal(err)
		}
		got, err := client.GetSecurityToken()
		if err != nil {
			t.Fatal(err)
		}
		if got == nil || *got != token {
			t.Fatal("Aliyun DNS token was not propagated")
		}
		tc := NewTencentYun(opts...).(*tencentYun)
		tcClient, err := tc.createClient()
		if err != nil {
			t.Fatal(err)
		}
		if tcClient.GetCredential().GetToken() != token {
			t.Fatal("Tencent DNS token was not propagated")
		}
	}
}
