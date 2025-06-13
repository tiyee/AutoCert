package applicant

type Options struct {
	Email           string `json:"email"`
	Domain          string `json:"domain"`
	AccessKeyId     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	Nameservers     string `json:"nameservers"`
}

type Option func(opts *Options)

// generate by z_gen

func (opts *Options) Update(opt ...Option) {
	for _, o := range opt {
		o(opts)
	}
}
func WithEmail(v string) Option {
	return func(opts *Options) {
		opts.Email = v
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
func WithNameservers(v string) Option {
	return func(opts *Options) {
		opts.Nameservers = v
	}
}
