package deployer

import (
	"github.com/tiyee/AutoCert/internal/applicant"
)

type Options struct {
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
