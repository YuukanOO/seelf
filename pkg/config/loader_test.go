package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/YuukanOO/seelf/pkg/config"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type (
	configuration struct {
		Verbose  bool                  `yaml:"verbose"`
		Http     httpConfiguration     `yaml:"http"`
		Balancer balancerConfiguration `yaml:"balancer"`
	}

	httpConfiguration struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port" env:"PORT"`
	}

	balancerConfiguration struct {
		Domain    string `yaml:"domain" env:"BALANCER_DOMAIN"`
		AcmeEmail string `yaml:"acme_email" env:"ACME_EMAIL"`
	}

	configurationWithProcessable struct {
		configuration
	}
)

var errPostLoad = errors.New("post load error")

func (*configurationWithProcessable) PostLoad() error {
	return errPostLoad
}

func Test_Load(t *testing.T) {
	var (
		confPath = filepath.Join("test_data", "test.conf")
		envFile  = ".env"
	)

	t.Cleanup(func() {
		os.Remove(envFile)
		os.RemoveAll("test_data")
	})

	if err := ostools.WriteFile(confPath, []byte(`verbose: true
http:
  host: 192.168.1.1
  port: 7777
`)); err != nil {
		t.Fatal(err)
	}

	if err := ostools.WriteFile(envFile, []byte(`BALANCER_DOMAIN=https://some.domain
ACME_EMAIL=admin@example.com
PORT=9999`)); err != nil {
		t.Fatal(err)
	}

	t.Run("should load the file from various sources", func(t *testing.T) {
		var conf configuration

		err := config.Load(confPath, &conf)

		testutil.IsNil(t, err)
		testutil.Equals(t, true, conf.Verbose)
		testutil.Equals(t, "192.168.1.1", conf.Http.Host)
		testutil.Equals(t, 9999, conf.Http.Port)
		testutil.Equals(t, "https://some.domain", conf.Balancer.Domain)
		testutil.Equals(t, "admin@example.com", conf.Balancer.AcmeEmail)
	})

	t.Run("should call the PostLoad method if the target implements the Processable interface", func(t *testing.T) {
		var conf configurationWithProcessable

		err := config.Load(confPath, &conf)

		testutil.ErrorIs(t, errPostLoad, err)
	})
}

func Test_Save(t *testing.T) {
	var (
		confPath = filepath.Join("test_data", "test.conf")
	)

	t.Cleanup(func() {
		os.RemoveAll("test_data")
	})

	t.Run("should save the configuration correctly", func(t *testing.T) {
		conf := configuration{
			Verbose: true,
			Http: httpConfiguration{
				Host: "127.0.0.1",
				Port: 8080,
			},
			Balancer: balancerConfiguration{
				Domain:    "http://localhost",
				AcmeEmail: "test@example.com",
			},
		}

		err := config.Save(confPath, conf)

		testutil.IsNil(t, err)
		b, err := os.ReadFile(confPath)
		testutil.IsNil(t, err)
		testutil.Equals(t, `verbose: true
http:
    host: 127.0.0.1
    port: 8080
balancer:
    domain: http://localhost
    acme_email: test@example.com
`, string(b))
	})
}
