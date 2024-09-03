package config_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/config"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/ostools"
)

type (
	configuration struct {
		Verbose  bool                  `yaml:"verbose"`
		Http     httpConfiguration     `yaml:"http"`
		Balancer balancerConfiguration `yaml:"balancer"`
	}

	httpConfiguration struct {
		Host    string            `yaml:"host"`
		Secure  monad.Maybe[bool] `yaml:"secure,omitempty" env:"HTTP_SECURE"`
		HttpTwo monad.Maybe[bool] `yaml:"http_two,omitempty" env:"HTTP_TWO"`
		Dummy   monad.Maybe[bool] `yaml:"dummy,omitempty" env:"HTTP_DUMMY"`
		Port    int               `yaml:"port" env:"PORT"`
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
	// Since for some tests, the monad has the initial value set to true but the
	// env removes it (setting the monad hasValue to false but keeping the initial value)
	unsetMonad := monad.Value(true)
	unsetMonad.Unset()

	tests := []struct {
		name     string
		conf     string
		env      string
		expected configuration
	}{
		{
			name: "configuration-only",
			conf: `verbose: true
http:
  host: 192.168.1.1
  port: 7777`,
			expected: configuration{
				Verbose: true,
				Http: httpConfiguration{
					Host: "192.168.1.1",
					Port: 7777,
				},
			},
		},
		{
			name: "configuration-and-env",
			conf: `verbose: true
http:
  host: 192.168.1.1
  port: 7777`,
			env: `BALANCER_DOMAIN=https://some.domain
ACME_EMAIL=admin@example.com
PORT=9999`,
			expected: configuration{
				Verbose: true,
				Http: httpConfiguration{
					Host: "192.168.1.1",
					Port: 9999,
				},
				Balancer: balancerConfiguration{
					Domain:    "https://some.domain",
					AcmeEmail: "admin@example.com",
				},
			},
		},
		{
			name: "env-only",
			env: `BALANCER_DOMAIN=https://some.domain
ACME_EMAIL=admin@example.com
PORT=9999`,
			expected: configuration{
				Http: httpConfiguration{
					Port: 9999,
				},
				Balancer: balancerConfiguration{
					Domain:    "https://some.domain",
					AcmeEmail: "admin@example.com",
				},
			},
		},
		{
			name: "conf-with-maybe",
			conf: `http:
  secure: true
  http_two: false`,
			expected: configuration{
				Http: httpConfiguration{Secure: monad.Value(true), HttpTwo: monad.Value(false)},
			},
		},
		{
			name: "env-with-maybe",
			env: `HTTP_SECURE=true
HTTP_TWO=false`,
			expected: configuration{
				Http: httpConfiguration{Secure: monad.Value(true), HttpTwo: monad.Value(false)},
			},
		},
		{
			name: "conf-and-env-with-maybe",
			conf: `http:
  secure: true
  http_two: false`,
			env: `HTTP_SECURE=
HTTP_TWO=true`,
			expected: configuration{
				Http: httpConfiguration{Secure: unsetMonad, HttpTwo: monad.Value(true)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confFilename := fmt.Sprintf("%s.yml", tt.name)
			envFilename := fmt.Sprintf(".%s.env", tt.name)

			t.Cleanup(func() {
				os.Remove(envFilename)
				os.Remove(confFilename)
			})

			os.Clearenv()

			if tt.conf != "" {
				err := ostools.WriteFile(confFilename, []byte(tt.conf))
				assert.Nil(t, err)
			}

			if tt.env != "" {
				err := ostools.WriteFile(envFilename, []byte(tt.env))
				assert.Nil(t, err)
			}

			var conf configuration

			exists, err := config.Load(confFilename, &conf, envFilename)
			assert.Nil(t, err)
			assert.Equal(t, tt.conf != "", exists)
			assert.DeepEqual(t, tt.expected, conf)
		})
	}

	t.Run("should call the PostLoad method if the target implements the Processable interface", func(t *testing.T) {
		var (
			conf         configurationWithProcessable
			confFilename = "test-conf.yml"
		)

		exists, err := config.Load(confFilename, &conf)

		assert.ErrorIs(t, errPostLoad, err)
		assert.False(t, exists)
	})
}

func Test_Save(t *testing.T) {
	confFilename := "test-conf.yml"

	t.Cleanup(func() {
		os.Remove(confFilename)
	})

	t.Run("should save the configuration correctly", func(t *testing.T) {
		conf := configuration{
			Verbose: true,
			Http: httpConfiguration{
				Host:    "127.0.0.1",
				Secure:  monad.Value(true),
				HttpTwo: monad.Value(false),
				Port:    8080,
			},
			Balancer: balancerConfiguration{
				Domain:    "http://localhost",
				AcmeEmail: "test@example.com",
			},
		}

		err := config.Save(confFilename, conf)

		assert.Nil(t, err)
		b, err := os.ReadFile(confFilename)
		assert.Nil(t, err)
		assert.Equal(t, `verbose: true
http:
    host: 127.0.0.1
    secure: true
    http_two: false
    port: 8080
balancer:
    domain: http://localhost
    acme_email: test@example.com
`, string(b))
	})
}
