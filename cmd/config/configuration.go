package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/YuukanOO/seelf/cmd/serve"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded"
	"github.com/YuukanOO/seelf/pkg/bytesize"
	"github.com/YuukanOO/seelf/pkg/config"
	"github.com/YuukanOO/seelf/pkg/crypto"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/arrays"
)

var (
	dotEnvFilenames          = []string{".env", ".env.local"}
	userConfigDir            = must.Panic(os.UserConfigDir())
	generatedSecretKey       = must.Panic(crypto.RandomKey[string](64))
	defaultDataDirectory     = filepath.Join(userConfigDir, "seelf")
	DefaultConfigPath        = filepath.Join(defaultDataDirectory, defaultConfigFilename) // Default configuration path
	noticeSecretKeyGenerated = `a default secret key has been generated. If you want to override it, you can set the HTTP_SECRET environment variable.`
)

const (
	databaseConnectionString            = "seelf.db?_journal=WAL&_timeout=5000&_foreign_keys=yes&_txlock=immediate&_synchronous=NORMAL"
	defaultConfigFilename               = "conf.yml"
	defaultPort                         = 8080
	defaultHost                         = ""
	defaultRunnersPollInterval          = "4s"
	defaultDeploymentWorkersCount uint8 = 4
	defaultGeneralWorkersCount    uint8 = 2
	defaultBalancerDomain               = "http://docker.localhost"
	defaultDeploymentDirTemplate        = "{{ .Environment }}"
	defaultSourceArchiveMaxSize         = "32mb"
)

type (
	// Configuration used to configure seelf commands exposed through an interface to prevent
	// direct access to internal configuration structure.
	Configuration interface {
		serve.Options

		Initialize(log.ConfigurableLogger, string) error // Initialize the configuration by loading it (from config file, env vars, etc.)
	}

	// Configuration builder function used to initialize the configuration object (mostly used in tests).
	ConfigurationBuilder func(*configuration)

	configuration struct {
		Log     logConfiguration
		Data    dataConfiguration
		Source  sourceConfiguration
		Http    httpConfiguration
		Runners runnersConfiguration  `env:"RUNNERS_CONFIGURATION"`
		Private internalConfiguration `yaml:"-"`
	}

	logConfiguration struct {
		Level  string `env:"LOG_LEVEL"`
		Format string `env:"LOG_FORMAT"`

		level  log.Level
		format log.OutputFormat
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

		deploymentDirTemplate *template.Template
	}

	sourceArchiveConfiguration struct {
		MaxSize string `env:"SOURCE_ARCHIVE_MAX_SIZE" yaml:"max_size"`

		maxSize int64
	}

	// Contains configuration related to deployment sources
	sourceConfiguration struct {
		Archive sourceArchiveConfiguration
	}

	// Configuration related to the background runners.
	runnerConfiguration struct {
		PollInterval string   `yaml:"poll_interval"`
		Count        uint8    `yaml:"count"`
		Jobs         []string `yaml:"jobs,omitempty"`

		pollInterval time.Duration
	}

	// internalConfiguration fields not read from the configuration file and use only during specific steps
	internalConfiguration struct {
		Email     string `env:"SEELF_ADMIN_EMAIL,ADMIN_EMAIL"`
		Password  string `env:"SEELF_ADMIN_PASSWORD,ADMIN_PASSWORD"`
		ExposedOn string `env:"EXPOSED_ON"` // Container name and default target url (ie. http://seelf@docker.localhost)

		exposedUrl monad.Maybe[domain.Url]
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
		Source: sourceConfiguration{
			Archive: sourceArchiveConfiguration{
				MaxSize: defaultSourceArchiveMaxSize,
			},
		},
		Http: httpConfiguration{
			Host:   defaultHost,
			Port:   defaultPort,
			Secret: generatedSecretKey,
		},
		Runners: defaultRunnersConfiguration(
			defaultRunnersPollInterval,
			defaultDeploymentWorkersCount,
			defaultGeneralWorkersCount,
		),
	}

	for _, builder := range builders {
		builder(conf)
	}

	if err := conf.validate(); err != nil {
		panic(err) // Should never happen since the default config is managed by us
	}

	return conf
}

func (c *configuration) Initialize(logger log.ConfigurableLogger, path string) error {
	var configFileFound bool

	if err := config.Load(c,
		config.FromYAML(path, &configFileFound),
		config.FromEnvironment(dotEnvFilenames...),
	); err != nil {
		return err
	}

	if err := c.validate(); err != nil {
		return err
	}

	// Make sure the data path exists
	if err := ostools.MkdirAll(c.Data.Path); err != nil {
		return err
	}

	// Update logger based on loaded configuration
	if err := logger.Configure(c.Log.format, c.Log.level); err != nil {
		return err
	}

	if configFileFound {
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

func (c *configuration) DataDir() string { return c.Data.Path }
func (c *configuration) DeploymentDirTemplate() *template.Template {
	return c.Data.deploymentDirTemplate
}
func (c *configuration) AppExposedUrl() monad.Maybe[domain.Url] { return c.Private.exposedUrl }
func (c *configuration) DefaultEmail() string                   { return c.Private.Email }
func (c *configuration) DefaultPassword() string                { return c.Private.Password }
func (c *configuration) Secret() []byte                         { return []byte(c.Http.Secret) }
func (c *configuration) IsDebug() bool                          { return c.Log.level == log.DebugLevel }
func (c *configuration) MaxDeploymentArchiveFileSize() int64 {
	return c.Source.Archive.maxSize
}

func (c *configuration) RunnersDefinitions(mapper *storage.DiscriminatedMapper[bus.AsyncRequest]) ([]embedded.RunnerDefinition, error) {
	definitions := make([]embedded.RunnerDefinition, len(c.Runners))
	unhandledMessages := mapper.Keys()

	for i, r := range c.Runners {
		// No specific messages set, handle all messages not seen already.
		// Since the validate function ensure only the last worker can have an empty list,
		// we should be good.
		if len(r.Jobs) == 0 {
			r.Jobs = unhandledMessages
			unhandledMessages = nil
		}

		messages := make([]bus.AsyncRequest, len(r.Jobs))

		for j, msg := range r.Jobs {
			// Remove the msg from the unhandledMessages
			msgIdx := slices.Index(unhandledMessages, msg)

			if msgIdx != -1 {
				unhandledMessages = slices.Delete(unhandledMessages, msgIdx, msgIdx+1)
			}

			req, err := mapper.From(msg, "{}")

			if err != nil {
				return nil, fmt.Errorf("unknown job name: %s, must be one of %s", msg, strings.Join(mapper.Keys(), ", "))
			}

			messages[j] = req
		}

		definitions[i] = embedded.RunnerDefinition{
			PollInterval: r.pollInterval,
			WorkersCount: r.Count,
			Messages:     messages,
		}
	}

	if len(unhandledMessages) > 0 {
		return nil, fmt.Errorf("some background jobs are not handled: %s, please fix your configuration by adding a worker to handle them", strings.Join(unhandledMessages, ", "))
	}

	return definitions, nil
}

func (c *configuration) IsSecure() bool {
	// If secure has been explicitly isSet, returns it
	if secure, isSet := c.Http.Secure.TryGet(); isSet {
		return secure
	}

	if defaultTargetUrl, isSet := c.Private.exposedUrl.TryGet(); isSet {
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

func (c *configuration) validate() error {
	lastRunnerIdx := len(c.Runners) - 1

	return validate.Struct(validate.Of{
		"log.level":                    validate.Value(c.Log.Level, &c.Log.level, log.ParseLevel),
		"log.format":                   validate.Value(c.Log.Format, &c.Log.format, log.ParseFormat),
		"source.archive.max_size":      validate.Value(c.Source.Archive.MaxSize, &c.Source.Archive.maxSize, bytesize.Parse),
		"data.deployment_dir_template": validate.Value(c.Data.DeploymentDirTemplate, &c.Data.deploymentDirTemplate, template.New("").Parse),
		"runners": validate.Array(c.Runners, func(runner runnerConfiguration, idx int) error {
			return validate.Struct(validate.Of{
				"poll_interval": validate.Value(runner.PollInterval, &c.Runners[idx].pollInterval, time.ParseDuration),
				"jobs": validate.If(idx != lastRunnerIdx, func() error {
					return validate.Field(runner.Jobs, arrays.Required)
				}),
			})
		}),
		"exposed_as": validate.If(c.Private.ExposedOn != "", func() error {
			url, err := domain.UrlFrom(c.Private.ExposedOn)

			if err != nil {
				return err
			}

			c.Private.exposedUrl.Set(url)

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
