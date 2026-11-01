package applicant

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/challenge/dns01"
)

type recordHandle struct{ zone, id string }
type recordLease struct {
	record recordHandle
	users  int
}

// ownedDNSProvider 仅按创建返回的 ID 清理，不重新查询同名 TXT。
// 创建失败（包括响应丢失）时不猜测记录 ID，宁可保留记录也不误删。
type ownedDNSProvider struct {
	mu      sync.Mutex
	records map[string]recordLease
	create  func(fqdn, value string) (recordHandle, error)
	remove  func(recordHandle) error
}

func (p *ownedDNSProvider) Timeout() (time.Duration, time.Duration) {
	return 200 * time.Second, 5 * time.Second
}
func leaseKey(domain, keyAuth string) string {
	return strings.ToLower(strings.TrimSuffix(domain, ".")) + "\x00" + keyAuth
}
func (p *ownedDNSProvider) Present(domain, token, keyAuth string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := leaseKey(domain, keyAuth)
	if lease, ok := p.records[key]; ok {
		lease.users++
		p.records[key] = lease
		return nil
	}
	info := dns01.GetChallengeInfo(domain, keyAuth)
	record, err := p.create(info.EffectiveFQDN, info.Value)
	if err != nil {
		return err
	}
	if record.id == "" {
		return fmt.Errorf("DNS API did not return a record ID")
	}
	if p.records == nil {
		p.records = make(map[string]recordLease)
	}
	p.records[key] = recordLease{record: record, users: 1}
	return nil
}
func (p *ownedDNSProvider) CleanUp(domain, token, keyAuth string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := leaseKey(domain, keyAuth)
	lease, ok := p.records[key]
	if !ok {
		return nil
	}
	if lease.users > 1 {
		lease.users--
		p.records[key] = lease
		return nil
	}
	if err := p.remove(lease.record); err != nil {
		return err
	}
	delete(p.records, key)
	return nil
}
func recordLocation(fqdn string) (string, string, error) {
	zone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return "", "", err
	}
	if strings.EqualFold(dns01.UnFqdn(fqdn), dns01.UnFqdn(zone)) {
		return dns01.UnFqdn(zone), "@", nil
	}
	rr, err := dns01.ExtractSubDomain(fqdn, zone)
	return dns01.UnFqdn(zone), rr, err
}
