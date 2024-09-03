package config

import (
	"os"
	"path"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/YuukanOO/seelf/cmd/serve"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/config"
	"github.com/YuukanOO/seelf/pkg/crypto"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/numbers"
)

var (
	userConfigDir            = must.Panic(os.UserConfigDir())
	generatedSecretKey       = must.Panic(crypto.RandomKey[string](64))
	defaultDataDirectory     = filepath.Join(userConfigDir, "seelf")
	DefaultConfigPath        = filepath.Join(defaultDataDirectory, defaultConfigFilename) // Default configuration path
	noticeSecretKeyGenerated = `a default secret key has been generated. If you want to override it, you can set the HTTP_SECRET environment variable.`
)

const (
	databaseConnectionString      = "seelf.db?_journal=WAL&_timeout=5000&_foreign_keys=yes&_txlock=immediate"
	defaultConfigFilename         = "conf.yml"
	defaultPort                   = 8080
	defaultHost                   = ""
	defaultRunnersPollInterval    = "4s"
	defaultRunnersDeploymentCount = 4
	defaultCleanupDeploymentCount = 2
	defaultBalancerDomain         = "http://docker.localhost"
	defaultDeploymentDirTemplate  = "{{ .Environment }}"
)

type (
	// Configuration used to configure seelf commands.
	Configuration interface {
		serve.Options // The configuration should provide every settings needed by the seelf server

		Initialize(log.ConfigurableLogger, string) error // Initialize the configuration by loading it (from config file, env vars, etc.)
	}

	// Configuration builder function used to initialize the configuration object (mostly used in tests).
	ConfigurationBuilder func(*configuration)

	configuration struct {
		Log     logConfiguration
		Data    dataConfiguration
		Http    httpConfiguration
		Runners runnersConfiguration
		Private internalConfiguration `yaml:"-"`

		appExposedUrl         monad.Maybe[domain.Url]
		pollInterval          time.Duration
		deploymentDirTemplate *template.Template
		logLevel              log.Level
		logFormat             log.OutputFormat
	}

	logConfiguration struct {
		Level  string `env:"LOG_LEVEL"`
		Format string `env:"LOG_FORMAT"`
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
		Cleanup      int    `env:"RUNNERS_CLEANUP_COUNT" yaml:"cleanup"`
	}

	// internalConfiguration fields not read from the configuration file and use only during specific steps
	internalConfiguration struct {
		Email     string `env:"SEELF_ADMIN_EMAIL,ADMIN_EMAIL"`
		Password  string `env:"SEELF_ADMIN_PASSWORD,ADMIN_PASSWORD"`
		ExposedOn string `env:"EXPOSED_ON"` // Container name and default target url (ie. http://seelf@docker.localhost)
	}
)

// Instantiate the default seelf configuration.
func Default(builders ...ConfigurationBuilder) Configuration {
	conf := &configuration{
		Log: logConfiguration{
			Level:  "info",
			Format: "console",
		},
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
			Cleanup:      defaultCleanupDeploymentCount,
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

func (c *configuration) Initialize(logger log.ConfigurableLogger, path string) error {
	exists, err := config.Load(path, c)

	if err != nil {
		return err
	}

	// Make sure the data path exists
	if err = ostools.MkdirAll(c.Data.Path); err != nil {
		return err
	}

	// Update logger based on loaded configuration
	if err = logger.Configure(c.logFormat, c.logLevel); err != nil {
		return err
	}

	if exists {
		logger.Infow("configuration loaded",
			"path", path)
	} else {
		logger.Infow("configuration not found, saving current configuration",
			"path", path)

		// Save the config file only if it doesn't exist yet (to preserve the secret generated for example)
		if err := config.Save(path, c); err != nil {
			return err
		}
	}

	if c.Http.Secret == generatedSecretKey {
		logger.Info(noticeSecretKeyGenerated)
	}

	return nil
}

func (c *configuration) DataDir() string                           { return c.Data.Path }
func (c *configuration) DeploymentDirTemplate() *template.Template { return c.deploymentDirTemplate }
func (c *configuration) AppExposedUrl() monad.Maybe[domain.Url]    { return c.appExposedUrl }
func (c *configuration) DefaultEmail() string                      { return c.Private.Email }
func (c *configuration) DefaultPassword() string                   { return c.Private.Password }
func (c *configuration) Secret() []byte                            { return []byte(c.Http.Secret) }
func (c *configuration) RunnersPollInterval() time.Duration        { return c.pollInterval }
func (c *configuration) RunnersDeploymentCount() int               { return c.Runners.Deployment }
func (c *configuration) RunnersCleanupCount() int                  { return c.Runners.Cleanup }

func (c *configuration) IsSecure() bool {
	// If secure has been explicitly isSet, returns it
	if secure, isSet := c.Http.Secure.TryGet(); isSet {
		return secure
	}

	if defaultTargetUrl, isSet := c.appExposedUrl.TryGet(); isSet {
		return defaultTargetUrl.UseSSL()
	}

	return false
}

// Gets the connection string to be used.
func (c *configuration) ConnectionString() string {
	return "file:" + path.Join(c.Data.Path, databaseConnectionString)
}

// Returns the address to bind the HTTP server to.
func (c *configuration) ListenAddress() string {
	return c.Http.Host + ":" + strconv.Itoa(c.Http.Port)
}

func (c *configuration) PostLoad() error {
	return validate.Struct(validate.Of{
		"log.level":                    validate.Value(c.Log.Level, &c.logLevel, log.ParseLevel),
		"log.format":                   validate.Value(c.Log.Format, &c.logFormat, log.ParseFormat),
		"data.deployment_dir_template": validate.Value(c.Data.DeploymentDirTemplate, &c.deploymentDirTemplate, template.New("").Parse),
		"runners.poll_interval":        validate.Value(c.Runners.PollInterval, &c.pollInterval, time.ParseDuration),
		"runners.deployment":           validate.Field(c.Runners.Deployment, numbers.Min(1)),
		"runners.cleanup":              validate.Field(c.Runners.Cleanup, numbers.Min(1)),
		"exposed_as": validate.If(c.Private.ExposedOn != "", func() error {
			url, err := domain.UrlFrom(c.Private.ExposedOn)

			if err != nil {
				return err
			}

			c.appExposedUrl.Set(url)

			return nil
		}),
	})
}

// Configuration builder used to set some tests sensible defaults.
// Generates a random data directory path to avoid conflicts with other tests.
func WithTestDefaults() ConfigurationBuilder {
	return func(c *configuration) {
		c.Data.Path = "__testdata_" + id.New[string]()
	}
}
