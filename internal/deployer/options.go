package deployer

import (
	"github.com/tiyee/AutoCert/internal/applicant"
)

type Options struct {
	SiteID          string                `json:"siteId"`
	Region          string                `json:"region"`
	SecurityToken   string                `json:"securityToken"`
	Domain          string                `json:"domain"`
	AccessKeyId     string                `json:"accessKeyId"`
	AccessKeySecret string                `json:"accessKeySecret"`
	Certificate     applicant.Certificate `json:"certificate"`
	Variables       map[string]string     `json:"variables"`
}

type Option func(opts *Options)

// generate by z_gen

func (opts *Options) Update(opt ...Option) {
	for _, o := range opt {
		o(opts)
	}
}
func WithDomain(v string) Option {
	return func(opts *Options) {
		opts.Domain = v
	}
}
func WithAccessKeyId(v string) Option {
	return func(opts *Options) {
		opts.AccessKeyId = v
	}
}
func WithAccessKeySecret(v string) Option {
	return func(opts *Options) {
		opts.AccessKeySecret = v
	}
}
func WithCertificate(v applicant.Certificate) Option {
	return func(opts *Options) {
		opts.Certificate = v
	}
}
func WithVariables(v map[string]string) Option {
	return func(opts *Options) {
		opts.Variables = v
	}
}

func WithSecurityToken(v string) Option {
	return func(opts *Options) { opts.SecurityToken = v }
}

func WithSiteID(v string) Option { return func(opts *Options) { opts.SiteID = v } }
func WithRegion(v string) Option { return func(opts *Options) { opts.Region = v } }
