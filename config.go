package webfw

import (
	"os"
	"reflect"

	"code.google.com/p/gcfg"
)

type Config struct {
	Server struct {
		Host     string
		Port     int
		CertFile string `gcfg:"cert-file"`
		KeyFile  string `gcfg:"key-file"`
		Devel    bool
	}
	Renderer struct {
		Base string
		Dir  string
	}
	Dispatcher struct {
		Middleware []string
	}
	Static struct {
		Dir      string
		Expires  string
		Prefix   string
		Index    string
		FileList bool `gcfg:"file-list"`
	}
	Session struct {
		Dir             string
		Secret          string
		MaxAge          string `gcfg:"max-age"`
		CleanupInterval string `gcfg:"cleanup-interval"`
		CleanupMaxAge   string `gcfg:"cleanup-max-age"`
	}
	I18n struct {
		Dir             string
		Languages       []string `gcfg:"language"`
		IgnoreURLPrefix []string `gcfg:"ignore-url-prefix"`
	}
}

// ReadConfig reads the given file path, merging it with the default
// configuration. If no path is given, the default configuration is returned.
// During the merge, the default configuration slice data is removed,
// thus only the provided configuration's slices are used.
func ReadConfig(path ...string) (Config, error) {
	def, err := defaultConfig()

	if err != nil {
		return Config{}, err
	}

	if len(path) == 0 {
		return def, nil
	}

	c := def
	clearSlices(&c)

	err = gcfg.ReadFileInto(&c, path[0])

	if err != nil {
		if os.IsNotExist(err) {
			return def, nil
		}

		return Config{}, err
	}

	return c, nil
}

// ParseConfig reads the given string, merging it with the default
// configuration. If no path is given, the default configuration is returned.
// During the merge, the default configuration slice data is removed,
// thus only the provided configuration's slices are used.
func ParseConfig(cfg ...string) (Config, error) {
	def, err := defaultConfig()

	if err != nil {
		return Config{}, err
	}

	if len(cfg) == 0 {
		return def, nil
	}

	c := def
	clearSlices(&c)

	err = gcfg.ReadStringInto(&c, cfg[0])

	if err != nil {
		return Config{}, err
	}

	return c, nil
}

func defaultConfig() (Config, error) {
	var def Config

	err := gcfg.ReadStringInto(&def, cfg)

	if err != nil {
		return Config{}, err
	}

	return def, nil
}

func clearSlices(dst *Config) {
	vDst := reflect.ValueOf(dst).Elem()

	deepClearSlices(vDst)
}

func deepClearSlices(dst reflect.Value) {
	if !dst.IsValid() {
		return
	}

	switch dst.Kind() {
	case reflect.Struct:
		for i, n := 0, dst.NumField(); i < n; i++ {
			deepClearSlices(dst.Field(i))
		}
	case reflect.Slice:
		if dst.CanSet() {
			tmp := reflect.MakeSlice(dst.Type(), 0, 0)
			dst.Set(tmp)
		}
	}
}

// Default configuration:
var cfg string = `
[server]
	host = ""
	port = 8080
	devel

[renderer]
	base = base.tmpl
	dir = templates

[dispatcher]
	middleware = Static
	middleware = Gzip
	middleware = Url # The uri mw has to be before the i18n
	middleware = I18N # The i18n mw has to be before the session
	middleware = Logger
	middleware = Session
	middleware = Context
	middleware = Error # should always be the last one wrapping middleware

[static]
	dir = static
	expires = 5m # 5 minutes

[session]
	dir = session
	secret = ___aVerySecr3tK3y&*7h4t5h0u1dR34l1yChaNg3!_=-
	max-age = 360h # 15 days
	cleanup-interval = 1h # 1 hour
	cleanup-max-age = 360h # 15 days

[i18n]
	dir = locale
`
