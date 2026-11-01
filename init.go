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

// InitCLI 解析参数；打印帮助、版本或配置错误后返回 false，入口无需继续处理。
func InitCLI() bool {
	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return false
	}
	if *version {
		v := "dev"
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
		fmt.Println("AutoCert", v)
		return false
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

	if Cfg.Domain == "" || Cfg.Email == "" {
		fmt.Fprintf(os.Stderr, "[%s] configuration: Domain or Email is required\n", Cfg.Domain)
		return false
	}
	return true
}
func ValidApplyCfg() bool {
	cfg := Cfg
	if cfg.DNSCredentials.AccessKeySecret == "" || cfg.DNSCredentials.AccessKeyId == "" {
		fmt.Fprintf(os.Stderr, "[%s] configuration: DNSAccessKeyID or DNSAccessKeySecret is required\n", cfg.Domain)
		return false
	}
	return true
}
func ValidDeployCfg() bool {
	cfg := Cfg
	if cfg.CertCredentials.AccessKeySecret == "" || cfg.CertCredentials.AccessKeyId == "" {
		fmt.Fprintf(os.Stderr, "[%s] configuration: CertAccessKeyID or CertAccessKeySecret is required\n", cfg.Domain)
		return false
	}
	return true
}
