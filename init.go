package AutoCert

import (
	"flag"
	"fmt"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/tiyee/AutoCert/internal/config"
	"github.com/tiyee/gokit/ptrlib"
	"os"
)

var Cfg config.Config
var (
	certOnly           = flag.Bool("certonly", false, "Obtain or renew a certificate, but do not install it")
	renew              = flag.Bool("renew", true, "Renew only one  previously obtained certificates that are near")
	version            = flag.Bool("version", false, "Show version and exit")
	domain             = flag.String("domain", "", "Domain name to obtain the certificate for")
	email              = flag.String("email", "", "Email address to obtain the certificate for")
	product            = flag.String("product", "", "Product name to obtain the certificate for")
	dnsAccessKeyId     = flag.String("dns-access-key-id", "", "dns Access Key ID")
	dnsAccessKeySecret = flag.String("dns-access-key-secret", "", "dns Access Key Secret")
	cdnAccessKeyId     = flag.String("cdn-access-key-id", "", "cdn Access Key ID")
	cdnAccessKeySecret = flag.String("cdn-access-key-secret", "", "cdn Access Key Secret")
)

func init() {

	flag.Parse()
	if !*certOnly {
		*renew = false
	}
	Cfg = config.Config{
		CertOnly: tea.BoolValue(certOnly),
		Renew:    tea.BoolValue(renew),
		Domain:   ptrlib.ToValue(domain, ""),
		Email:    ptrlib.ToValue(email, ""),
		Product:  ptrlib.ToValue(product, ""),
		DNSCredentials: config.ICredentials{
			AccessKeyId:     ptrlib.ToValue(dnsAccessKeyId, ""),
			AccessKeySecret: ptrlib.ToValue(dnsAccessKeySecret, ""),
		},
		CertCredentials: config.ICredentials{
			AccessKeyId:     ptrlib.ToValue(cdnAccessKeyId, ""),
			AccessKeySecret: ptrlib.ToValue(cdnAccessKeySecret, ""),
		},
	}
	if len(Cfg.Domain)*len(Cfg.Email) == 0 {
		fmt.Println("Domain or Email is required")
		os.Exit(1)
		return
	}
	fmt.Println(Cfg)

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
