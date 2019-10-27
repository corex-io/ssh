package ssh

// Options options
type Options struct {
	Name     string            `json:"name" yaml:"name"`
	Hostname string            `json:"hostname" yaml:"hostname"`
	IP       string            `json:"ip" yaml:"ip"`
	Port     int               `json:"port" yaml:"port"`
	Username string            `json:"username" yaml:"username"`
	Password string            `json:"password" yaml:"password"`
	Key      string            `json:"key" yaml:"key"`
	QAs      map[string]string `json:"qas" yaml:"qas"`
	Pseudo   bool              `json:"pseudo" yaml:"pseudo"` // like "ssh -tt", Force pseudo-terminal allocation.
	Timeout  int               `json:"timeout" yaml:"timeout"`
	Env      map[string]string `json:"env" yaml:"env"`
}

// Option func
type Option func(*Options)

func newOptions(opts ...Option) Options {
	opt := Options{
		Username: "root",
		Port:     22,
		QAs:      make(map[string]string),
		Timeout:  3,
		Env: map[string]string{
			"LANG": "zh_CN.UTF-8",
		},
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// Name set name
func Name(name string) Option {
	return func(o *Options) {
		o.Name = name
	}
}

// Hostname set hostname
func Hostname(hostname string) Option {
	return func(o *Options) {
		o.Hostname = hostname
	}
}

// IP set ip
func IP(ip string) Option {
	return func(o *Options) {
		o.IP = ip
	}
}

// Port set port
func Port(port int) Option {
	return func(o *Options) {
		o.Port = port
	}
}

// Username set username
func Username(username string) Option {
	return func(o *Options) {
		o.Username = username
	}
}

// Password set password
func Password(password string) Option {
	return func(o *Options) {
		o.Password = password
	}
}

// Key set key
func Key(key string) Option {
	return func(o *Options) {
		o.Key = key
	}
}

// QA set QA
func QA(key, value string) Option {
	return func(o *Options) {
		o.QAs[key] = value
	}
}

// Pseudo set pseudo
func Pseudo(pseudo bool) Option {
	return func(o *Options) {
		o.Pseudo = pseudo
	}
}

// Timeout set timeout
func Timeout(timeout int) Option {
	return func(o *Options) {
		o.Timeout = timeout
	}
}

// Env env
func Env(key, value string) Option {
	return func(o *Options) {
		o.Env[key] = value
	}
}
