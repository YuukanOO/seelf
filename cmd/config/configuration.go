package config

import (
	"errors"
	"fmt"
	"io/fs"
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
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/numbers"
)

var (
	userConfigDir                          = must.Panic(os.UserConfigDir())
	generatedSecretKey                     = must.Panic(crypto.RandomKey[string](64))
	defaultDataDirectory                   = filepath.Join(userConfigDir, "seelf")
	DefaultConfigPath                      = filepath.Join(defaultDataDirectory, defaultConfigFilename)
	configFingerprintName                  = "last_run_data"
	noticeNotSupportedConfigChangeDetected = `looks like you have changed the domain used by seelf for your apps (either the protocol or the domain itself).
	
	Those changes are not supported yet. For now, for things to keep running correctly, you'll have to manually redeploy all of your apps.`
	noticeSecretKeyGenerated = `a default secret key has been generated. If you want to override it, you can set the HTTP_SECRET environment variable.`
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

		Initialize(log.ConfigurableLogger, CliOptions) error // Initialize the configuration by loading it (from config file, env vars, etc.)
	}

	// Configuration builder function used to initialize the configuration object (mostly used in tests).
	ConfigurationBuilder func(*configuration)

	// Represents options sets on the CLI level.
	CliOptions struct {
		Path    string
		Verbose bool
	}

	configuration struct {
		Data     dataConfiguration
		Http     httpConfiguration
		Balancer balancerConfiguration
		Runners  runnersConfiguration
		Private  internalConfiguration `yaml:"-"`

		domain                domain.Url
		pollInterval          time.Duration
		deploymentDirTemplate *template.Template
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

func (c *configuration) Initialize(logger log.ConfigurableLogger, opts CliOptions) error {
	if opts.Path == "" {
		opts.Path = DefaultConfigPath
	}

	exists, err := config.Load(opts.Path, c)

	if err != nil {
		return err
	}

	// Update logger based on loaded configuration
	if opts.Verbose {
		logger.Configure(log.OutputConsole, log.DebugLevel, true)
	} else {
		logger.Configure(log.OutputConsole, log.InfoLevel, false)
	}

	if exists {
		logger.Infow("configuration loaded",
			"path", opts.Path)
	} else {
		logger.Infow("configuration not found, saving current configuration",
			"path", opts.Path)

		// Save the config file only if it doesn't exist yet (to preserve the secret generated for example)
		if err := config.Save(opts.Path, c); err != nil {
			return err
		}
	}

	if c.Http.Secret == generatedSecretKey {
		logger.Info(noticeSecretKeyGenerated)
	}

	return c.checkNonSupportedConfigChanges(logger)
}

func (c configuration) DataDir() string                           { return c.Data.Path }
func (c configuration) DeploymentDirTemplate() *template.Template { return c.deploymentDirTemplate }
func (c configuration) Domain() domain.Url                        { return c.domain }
func (c configuration) DefaultEmail() string                      { return c.Private.Email }
func (c configuration) DefaultPassword() string                   { return c.Private.Password }
func (c configuration) Secret() []byte                            { return []byte(c.Http.Secret) }
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

func (c *configuration) checkNonSupportedConfigChanges(logger log.Logger) error {
	fingerprintPath := filepath.Join(c.Data.Path, configFingerprintName)
	fingerprint := c.Balancer.Domain

	data, err := os.ReadFile(fingerprintPath)

	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	strdata := string(data)

	if strdata != "" && strdata != fingerprint {
		logger.Warn(noticeNotSupportedConfigChangeDetected)
	}

	return ostools.WriteFile(fingerprintPath, []byte(fingerprint))
}

// Configuration builder used to set some tests sensible defaults.
// Generates a random data directory path to avoid conflicts with other tests.
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
