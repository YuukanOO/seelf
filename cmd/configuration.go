package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/crypto"
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
	logsDir                       = "logs"
	appsDir                       = "apps"
)

type (
	// Structure which hold the application wide configuration.
	// Implements various options used throughout the application.
	configuration struct {
		Verbose  bool `env:"SEELF_DEBUG" yaml:",omitempty"`
		Data     dataConfiguration
		Http     httpConfiguration
		Balancer balancerConfiguration
		Runners  runnersConfiguration
		Private  internalConfiguration `yaml:"-"`

		currentVersion        string
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

func defaultConfiguration() *configuration {
	conf := &configuration{
		path:           filepath.Join(defaultDataDirectory, defaultConfigFilename),
		currentVersion: currentVersion(),
		Verbose:        false,
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

	if err := conf.PostLoad(); err != nil {
		panic(err) // Should never happen since the default config is managed by us
	}

	return conf
}

func (c configuration) DataDir() string                                     { return c.Data.Path }
func (c configuration) DeploymentDirTemplate() domain.DeploymentDirTemplate { return c }
func (c configuration) AppsDir() string                                     { return filepath.Join(c.DataDir(), appsDir) }
func (c configuration) LogsDir() string                                     { return filepath.Join(c.DataDir(), logsDir) }
func (c configuration) Domain() domain.Url                                  { return c.domain }
func (c configuration) DefaultEmail() string                                { return c.Private.Email }
func (c configuration) DefaultPassword() string                             { return c.Private.Password }
func (c configuration) Secret() []byte                                      { return []byte(c.Http.Secret) }
func (c configuration) IsUsingGeneratedSecret() bool                        { return c.Http.Secret == generatedSecretKey }
func (c configuration) IsVerbose() bool                                     { return c.Verbose }
func (c configuration) ConfigPath() string                                  { return c.path }
func (c configuration) CurrentVersion() string                              { return c.currentVersion }
func (c configuration) RunnersPollInterval() time.Duration                  { return c.pollInterval }
func (c configuration) RunnersDeploymentCount() int                         { return c.Runners.Deployment }

func (c configuration) IsSecure() bool {
	// If secure has been explicitly set, returns it
	if c.Http.Secure.HasValue() {
		return c.Http.Secure.MustGet()
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
		"runners.deployment":           validation.Is(c.Runners.Deployment, numbers.Min(0)),
		"balancer.domain":              domainUrlErr,
		"balancer.acme.email": validation.If(domainUrlErr == nil && c.domain.UseSSL(), func() error {
			return validation.Value(c.AcmeEmail(), &acmeEmail, auth.EmailFrom)
		}),
	})
}

func (c configuration) Execute(data domain.DeploymentTemplateData) string {
	var w strings.Builder

	if err := c.deploymentDirTemplate.Execute(&w, data); err != nil {
		panic(err)
	}

	return w.String()
}
