package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"text/template"
	"time"

	"github.com/YuukanOO/seelf/cmd/serve"
	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/config"
	"github.com/YuukanOO/seelf/pkg/crypto"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/numbers"
)

var (
	userConfigDir        = must.Panic(os.UserConfigDir())
	generatedSecretKey   = must.Panic(crypto.RandomKey[string](64))
	defaultDataDirectory = filepath.Join(userConfigDir, "seelf")
)

const (
	databaseFilename              = "seelf.db?_foreign_keys=yes&_txlock=immediate"
	defaultConfigFilename         = "conf.yml"
	defaultPort                   = 8080
	defaultHost                   = ""
	defaultRunnersPollInterval    = "4s"
	defaultRunnersDeploymentCount = 4
	defaultBalancerDomain         = "http://docker.localhost"
	defaultDeploymentDirTemplate  = "{{ .Environment }}"
)

type (
	// Configuration used to configure seelf commands.
	Configuration interface {
		serve.Options // The configuration should provide every settings needed by the seelf server

		Initialize(path string, verbose bool) error // Initialize the configuration by loading it (from config file, env vars, etc.)
	}

	// Configuration builder function used to initialize the configuration object (mostly used in tests).
	ConfigurationBuilder func(*configuration)

	configuration struct {
		Verbose  bool `env:"SEELF_DEBUG" yaml:",omitempty"`
		Data     dataConfiguration
		Http     httpConfiguration
		Balancer balancerConfiguration
		Runners  runnersConfiguration
		Private  internalConfiguration `yaml:"-"`

		domain                domain.Url
		pollInterval          time.Duration
		deploymentDirTemplate *template.Template
		path                  string // Holds from where the config was loaded / saved
	}

	httpConfiguration struct {
		Host   string            `env:"HTTP_HOST" yaml:",omitempty"`
		Port   int               `env:"HTTP_PORT,PORT"`
		Secure monad.Maybe[bool] `env:"HTTP_SECURE" yaml:",omitempty"`
		Secret string            `env:"HTTP_SECRET"`
	}

	// Contains configuration related to where files produced by seelf will be stored.
	dataConfiguration struct {
		Path                  string `env:"DATA_PATH"`
		DeploymentDirTemplate string `env:"DEPLOYMENT_DIR_TEMPLATE" yaml:"deployment_dir_template"`
	}

	// Configuration related to the async jobs runners.
	runnersConfiguration struct {
		PollInterval string `env:"RUNNERS_POLL_INTERVAL" yaml:"poll_interval"`
		Deployment   int    `env:"RUNNERS_DEPLOYMENT_COUNT" yaml:"deployment"`
	}

	// internalConfiguration fields not read from the configuration file and use only during specific steps
	internalConfiguration struct {
		Email    string `env:"SEELF_ADMIN_EMAIL"`
		Password string `env:"SEELF_ADMIN_PASSWORD"`
	}

	// Contains configuration related to the balancerConfiguration.
	balancerConfiguration struct {
		Domain string            `env:"BALANCER_DOMAIN"`
		Acme   acmeConfiguration `yaml:",omitempty"`
	}

	acmeConfiguration struct {
		Email string `env:"ACME_EMAIL" yaml:",omitempty"`
	}
)

// Instantiate the default seelf configuration.
func Default(builders ...ConfigurationBuilder) Configuration {
	conf := &configuration{
		path:    filepath.Join(defaultDataDirectory, defaultConfigFilename),
		Verbose: false,
		Data: dataConfiguration{
			Path:                  defaultDataDirectory,
			DeploymentDirTemplate: defaultDeploymentDirTemplate,
		},
		Http: httpConfiguration{
			Host:   defaultHost,
			Port:   defaultPort,
			Secret: generatedSecretKey,
		},
		Runners: runnersConfiguration{
			PollInterval: defaultRunnersPollInterval,
			Deployment:   defaultRunnersDeploymentCount,
		},
		Balancer: balancerConfiguration{
			Domain: defaultBalancerDomain,
		},
	}

	for _, builder := range builders {
		builder(conf)
	}

	if err := conf.PostLoad(); err != nil {
		panic(err) // Should never happen since the default config is managed by us
	}

	return conf
}

func (c *configuration) Initialize(path string, verbose bool) error {
	c.path = path
	c.Verbose = verbose

	// FIXME: Maybe it could be a good idea to ask for a global logger to at least inform the user
	// that a config file has been read and/or created.

	exists, err := config.Load(c.path, c)

	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	// Save the config file only if it doesn't exist yet (to preserve the secret generated for example)
	return config.Save(c.path, c)
}

func (c configuration) DataDir() string                           { return c.Data.Path }
func (c configuration) DeploymentDirTemplate() *template.Template { return c.deploymentDirTemplate }
func (c configuration) Domain() domain.Url                        { return c.domain }
func (c configuration) DefaultEmail() string                      { return c.Private.Email }
func (c configuration) DefaultPassword() string                   { return c.Private.Password }
func (c configuration) Secret() []byte                            { return []byte(c.Http.Secret) }
func (c configuration) IsUsingGeneratedSecret() bool              { return c.Http.Secret == generatedSecretKey }
func (c configuration) IsVerbose() bool                           { return c.Verbose }
func (c configuration) ConfigPath() string                        { return c.path }
func (c configuration) RunnersPollInterval() time.Duration        { return c.pollInterval }
func (c configuration) RunnersDeploymentCount() int               { return c.Runners.Deployment }

func (c configuration) IsSecure() bool {
	// If secure has been explicitly isSet, returns it
	if secure, isSet := c.Http.Secure.TryGet(); isSet {
		return secure
	}

	// Else, fallback to the domain SSL value
	return c.domain.UseSSL()
}

func (c configuration) AcmeEmail() string {
	if c.Balancer.Acme.Email == "" {
		return c.DefaultEmail()
	}

	return c.Balancer.Acme.Email
}

// Gets the connection string to be used.
func (c configuration) ConnectionString() string {
	return fmt.Sprintf("file:%s", path.Join(c.Data.Path, databaseFilename))
}

// Returns the address to bind the HTTP server to.
func (c configuration) ListenAddress() string {
	return fmt.Sprintf("%s:%d", c.Http.Host, c.Http.Port)
}

func (c *configuration) PostLoad() error {
	var (
		acmeEmail    auth.Email
		domainUrlErr = validation.Value(c.Balancer.Domain, &c.domain, domain.UrlFrom)
	)

	return validation.Check(validation.Of{
		"data.deployment_dir_template": validation.Value(c.Data.DeploymentDirTemplate, &c.deploymentDirTemplate, template.New("").Parse),
		"runners.poll_interval":        validation.Value(c.Runners.PollInterval, &c.pollInterval, time.ParseDuration),
		"runners.deployment":           validation.Is(c.Runners.Deployment, numbers.Min(1)),
		"balancer.domain":              domainUrlErr,
		"balancer.acme.email": validation.If(domainUrlErr == nil && c.domain.UseSSL(), func() error {
			return validation.Value(c.AcmeEmail(), &acmeEmail, auth.EmailFrom)
		}),
	})
}

// Configuration builder used to set some tests sensible defaults.
func WithTestDefaults() ConfigurationBuilder {
	return func(c *configuration) {
		c.Data.Path = fmt.Sprintf("__testdata_%s", id.New[string]())
	}
}

// Configure the balancer for the given domain and acme email.
func WithBalancer(domain, acmeEmail string) ConfigurationBuilder {
	return func(c *configuration) {
		c.Balancer.Domain = domain
		c.Balancer.Acme.Email = acmeEmail
	}
}
