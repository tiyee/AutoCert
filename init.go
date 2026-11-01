package AutoCert

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/tiyee/gokit/ptrlib"

	"github.com/tiyee/AutoCert/internal/config"
)

var Cfg config.Config
var (
	dnsSiteID          = flag.String("dns-site-id", "", "Optional ESA site ID or TEO zone ID for DNS")
	cdnSiteID          = flag.String("cdn-site-id", "", "Optional ESA site ID or TEO zone ID for deployment")
	dnsRegion          = flag.String("dns-region", "", "Optional DNS API region (ESA defaults to cn-hangzhou)")
	cdnRegion          = flag.String("cdn-region", "", "Optional deployment API region (ESA defaults to cn-hangzhou)")
	certOnly           = flag.Bool("certonly", false, "Obtain a new certificate and deploy it (legacy certonly behavior)")
	renew              = flag.Bool("renew", true, "Compatibility flag; renew unless certonly is set")
	version            = flag.Bool("version", false, "Show version and exit")
	domain             = flag.String("domain", "", "Domain name to obtain the certificate for")
	email              = flag.String("email", "", "Email address to obtain the certificate for")
	product            = flag.String("product", "", "Product name to obtain the certificate for")
	dnsSecurityToken   = flag.String("dns-security-token", "", "Optional DNS STS security/session token")
	cdnSecurityToken   = flag.String("cdn-security-token", "", "Optional certificate STS security/session token")
	dnsAccessKeyId     = flag.String("dns-access-key-id", "", "dns Access Key ID")
	dnsAccessKeySecret = flag.String("dns-access-key-secret", "", "dns Access Key Secret")
	cdnAccessKeyId     = flag.String("cdn-access-key-id", "", "cdn Access Key ID")
	cdnAccessKeySecret = flag.String("cdn-access-key-secret", "", "cdn Access Key Secret")
	platform           = flag.String("platform", "aliyun", "aliyun, tencent, esa or teo")
	cdnPlatform        = flag.String("cdn-platform", "", "platform")
	dnsPlatform        = flag.String("dns-platform", "", "platform")
)

// InitCLI 由命令入口调用，导入包不解析参数或退出进程。
func InitCLI() {
	flag.Parse()
	if *version {
		v := "dev"
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
		fmt.Println("AutoCert", v)
		os.Exit(0)
	}
	if !*certOnly {
		*renew = false
	}
	Cfg = config.Config{
		DNSSiteID: *dnsSiteID, CDNSiteID: *cdnSiteID, DNSRegion: *dnsRegion, CDNRegion: *cdnRegion,
		Platform:    ptrlib.ToValue(platform, ""),
		CDNPlatform: ptrlib.ToValue(cdnPlatform, ""),
		DNSPlatform: ptrlib.ToValue(dnsPlatform, ""),
		CertOnly:    tea.BoolValue(certOnly),
		Renew:       tea.BoolValue(renew),
		Domain:      ptrlib.ToValue(domain, ""),
		Email:       ptrlib.ToValue(email, ""),
		Product:     ptrlib.ToValue(product, ""),
		DNSCredentials: config.ICredentials{
			SecurityToken:   *dnsSecurityToken,
			AccessKeyId:     ptrlib.ToValue(dnsAccessKeyId, ""),
			AccessKeySecret: ptrlib.ToValue(dnsAccessKeySecret, ""),
		},
		CertCredentials: config.ICredentials{
			SecurityToken:   *cdnSecurityToken,
			AccessKeyId:     ptrlib.ToValue(cdnAccessKeyId, ""),
			AccessKeySecret: ptrlib.ToValue(cdnAccessKeySecret, ""),
		},
	}
	Cfg = Cfg.WithDefaults()

	if len(Cfg.Domain)*len(Cfg.Email) == 0 {
		fmt.Println("Domain or Email is required")
		os.Exit(1)
		return
	}

}
func ValidApplyCfg() {
	cfg := Cfg
	if len(cfg.DNSCredentials.AccessKeySecret)*len(cfg.DNSCredentials.AccessKeyId) == 0 {
		fmt.Println("DNSAccessKeyID or DNSAccessKeySecret is required")
		os.Exit(1)
		return
	}
}
func ValidDeployCfg() {
	cfg := Cfg
	if len(cfg.CertCredentials.AccessKeySecret)*len(cfg.CertCredentials.AccessKeyId) == 0 {
		fmt.Println("CertAccessKeyID or CertAccessKeySecret is required")
		os.Exit(1)
		return
	}
}
