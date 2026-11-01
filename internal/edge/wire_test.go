package edge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	esa "github.com/alibabacloud-go/esa-20240910/v3/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ssl "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"
	teo "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/teo/v20220901"
)

// 使用真实 SDK 和本地 HTTP 服务检查序列化与 STS 签名，所有地址固定为 httptest 本机端点。
func TestESASDKWire(t *testing.T) {
	public, key, end := testCertificate(t, "www.example.com")
	var mu sync.Mutex
	counts := map[string]int{}
	injectError := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error("invalid form")
		}
		action := r.Form.Get("Action")
		if action == "" {
			action = r.Header.Get("X-Acs-Action")
		}
		mu.Lock()
		counts[action]++
		fail := injectError
		mu.Unlock()
		if r.Form.Get("SecurityToken") != "fake-sts" && r.Header.Get("X-Acs-Security-Token") != "fake-sts" {
			t.Error("STS token missing on ESA request")
		}
		w.Header().Set("Content-Type", "application/json")
		var body any
		switch action {
		case "ListSites":
			body = map[string]any{"TotalCount": 1, "Sites": []any{map[string]any{"SiteId": 1, "SiteName": "example.com", "AccessType": "NS"}}}
		case "CreateRecord":
			if fail {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{"Code": "Forbidden", "Message": "private-content"})
				return
			}
			var data map[string]string
			if json.Unmarshal([]byte(r.Form.Get("Data")), &data) != nil || data["Value"] != "fake-validation" || r.Form.Get("Type") != "TXT" || r.Form.Get("RecordName") != "_acme-challenge.example.com" {
				t.Error("incorrect serialized ESA TXT data")
			}
			body = map[string]any{"RecordId": 101}
		case "DeleteRecord":
			if r.Form.Get("RecordId") != "101" {
				t.Error("wrong ESA deletion ID")
			}
			body = map[string]any{}
		case "ListCertificatesByRecord":
			body = map[string]any{"Result": []any{map[string]any{"RecordName": "www.example.com", "Count": 1, "Certificates": []any{map[string]any{"Id": "bound", "Name": "third-party", "Type": "upload", "Status": "OK"}}}}}
		case "GetCertificate":
			body = map[string]any{"Certificate": public}
		case "SetCertificate":
			if r.Form.Get("Certificate") != public || r.Form.Get("PrivateKey") != key || r.Form.Get("Type") != "upload" || r.Form.Get("Id") != "" {
				t.Error("incorrect serialized ESA certificate")
			}
			body = map[string]any{"Id": "new-id"}
		default:
			t.Error("unexpected ESA action")
			body = map[string]any{}
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()
	c, err := NewESA("fake-id", "fake-secret", "fake-sts", "", "1")
	if err != nil {
		t.Fatal(err)
	}
	sdk := c.client.(*esa.Client)
	sdk.Endpoint = tea.String(strings.TrimPrefix(server.URL, "http://"))
	sdk.Protocol = tea.String("HTTP")
	record, err := c.CreateTXT(context.Background(), "_acme-challenge.example.com", "fake-validation")
	if err != nil {
		t.Fatal(err)
	}
	if err = c.DeleteTXT(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	certs, err := c.Certificates(context.Background(), "www.example.com")
	if err != nil || len(certs) != 1 || !certs[0].Expires.Equal(end) {
		t.Fatal("ESA real SDK certificate decoding failed", err)
	}
	if err = c.Deploy(context.Background(), "www.example.com", public, key); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	injectError = true
	mu.Unlock()
	if _, err = c.CreateTXT(context.Background(), "_acme-challenge.example.com", "fake-validation"); err == nil || !strings.Contains(err.Error(), "Forbidden") || strings.Contains(err.Error(), "private-content") {
		t.Fatal("ESA SDK error code lost or response message leaked")
	}
	mu.Lock()
	defer mu.Unlock()
	if counts["CreateRecord"] != 2 {
		t.Fatal("failed DNS write was retried")
	}
	for _, action := range []string{"ListSites", "CreateRecord", "DeleteRecord", "ListCertificatesByRecord", "GetCertificate", "SetCertificate"} {
		if counts[action] == 0 {
			t.Errorf("action not exercised: %s", action)
		}
	}
}
func TestTEOSDKWire(t *testing.T) {
	public, key, _ := testCertificate(t, "www.example.com")
	var mu sync.Mutex
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.Header.Get("X-TC-Action")
		mu.Lock()
		actions = append(actions, action)
		mu.Unlock()
		if r.Header.Get("X-TC-Token") != "fake-sts" {
			t.Error("STS token missing on Tencent request")
		}
		var req map[string]any
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			t.Error("invalid JSON request")
		}
		w.Header().Set("Content-Type", "application/json")
		var body any
		switch action {
		case "DescribeZones":
			body = map[string]any{"TotalCount": 1, "Zones": []any{map[string]any{"ZoneId": "zone-one", "ZoneName": "example.com", "Type": "full"}}}
		case "CreateDnsRecord":
			if req["Name"] != "_acme-challenge.example.com" || req["Content"] != "fake-validation" || req["ZoneId"] != "zone-one" {
				t.Error("incorrect serialized TEO TXT data")
			}
			body = map[string]any{"RecordId": "record-own"}
		case "DeleteDnsRecords":
			ids, _ := req["RecordIds"].([]any)
			if len(ids) != 1 || ids[0] != "record-own" {
				t.Error("wrong TEO deletion IDs")
			}
			body = map[string]any{}
		case "DescribeAccelerationDomains":
			body = map[string]any{"TotalCount": 1, "AccelerationDomains": []any{map[string]any{"DomainName": "www.example.com", "Certificate": map[string]any{"Mode": "sslcert", "List": []any{map[string]any{"CertId": "old", "Status": "deployed", "ExpireTime": "2027-01-01T00:00:00Z"}}}}}}
		case "UploadCertificate":
			if req["CertificatePublicKey"] != public || req["CertificatePrivateKey"] != key {
				t.Error("incorrect serialized SSL upload")
			}
			body = map[string]any{"CertificateId": "uploaded"}
		case "ModifyHostsCertificate":
			hosts, _ := req["Hosts"].([]any)
			certs, _ := req["ServerCertInfo"].([]any)
			if req["Mode"] != "sslcert" || req["ZoneId"] != "zone-one" || len(hosts) != 1 || hosts[0] != "www.example.com" || len(certs) != 1 {
				t.Error("incorrect serialized TEO binding")
			}
			if len(certs) == 1 {
				cert, _ := certs[0].(map[string]any)
				if cert["CertId"] != "uploaded" {
					t.Error("wrong certificate bound")
				}
			}
			body = map[string]any{}
		default:
			t.Error("unexpected Tencent action")
			body = map[string]any{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"Response": body})
	}))
	defer server.Close()
	c, err := NewTEO("fake-id", "fake-secret", "fake-sts", "", "zone-one")
	if err != nil {
		t.Fatal(err)
	}
	p := profile.NewClientProfile()
	p.HttpProfile.Scheme = "HTTP"
	p.HttpProfile.Endpoint = strings.TrimPrefix(server.URL, "http://")
	c.client.(*teo.Client).WithProfile(p)
	c.ssl.(*ssl.Client).WithProfile(p)
	record, err := c.CreateTXT(context.Background(), "_acme-challenge.example.com", "fake-validation")
	if err != nil {
		t.Fatal(err)
	}
	if err = c.DeleteTXT(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Certificates(context.Background(), "www.example.com"); err != nil {
		t.Fatal(err)
	}
	if err = c.Deploy(context.Background(), "www.example.com", public, key); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(actions) < 2 || actions[len(actions)-2] != "UploadCertificate" || actions[len(actions)-1] != "ModifyHostsCertificate" {
		t.Fatal("SDK upload/bind order mismatch")
	}
}
