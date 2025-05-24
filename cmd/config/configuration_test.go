package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YuukanOO/seelf/cmd/config"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	someOtherMessage = message{name: "some.other.message"}
	anotherMessage   = message{name: "and.another.message"}
	deployCommand    = message{name: "deployment.command.deploy"}
)

func Test_Configuration(t *testing.T) {

	t.Run("should correctly load a configuration from a yaml file", func(t *testing.T) {
		mapper := buildMapper(someOtherMessage, anotherMessage, deployCommand)
		logger, conf, err := loadConfiguration("valid-config.yml", nil)

		assert.Nil(t, err)
		assert.Equal(t, log.DebugLevel, logger.level)
		assert.Equal(t, log.OutputJSON, logger.format)
		assert.True(t, conf.IsDebug())
		assert.True(t, conf.IsSecure())
		assert.Equal(t, "localhost:5000", conf.ListenAddress())
		assert.Equal(t, "file:testdata/seelf.db?_journal=WAL&_timeout=5000&_foreign_keys=yes&_txlock=immediate&_synchronous=NORMAL", conf.ConnectionString())
		assert.Equal(t, "testdata", conf.DataDir())
		assert.Equal(t, 10485760, conf.MaxDeploymentArchiveFileSize())
		assert.DeepEqual(t, []byte("top_secret"), conf.Secret())

		runners, err := conf.RunnersDefinitions(mapper)
		assert.Nil(t, err)
		assert.HasLength(t, 2, runners)

		runner := runners[0]
		assert.Equal(t, 5*time.Second, runner.PollInterval)
		assert.Equal(t, 4, runner.WorkersCount)
		assert.DeepEqual(t, []bus.AsyncRequest{
			deployCommand,
		}, runner.Messages)

		runner = runners[1]
		assert.Equal(t, 10*time.Second, runner.PollInterval)
		assert.Equal(t, 5, runner.WorkersCount)
		assert.ArrayEqualFunc(t, []bus.AsyncRequest{
			someOtherMessage,
			anotherMessage,
		}, runner.Messages, compareRequestFunc)
	})

	t.Run("should correctly handle deprecated runners configuration", func(t *testing.T) {
		mapper := buildMapper(someOtherMessage, anotherMessage, deployCommand)
		_, conf, err := loadConfiguration("deprecated-runners-config.yml", nil)

		assert.Nil(t, err)

		runners, err := conf.RunnersDefinitions(mapper)
		assert.Nil(t, err)
		assert.HasLength(t, 2, runners)

		runner := runners[0]
		assert.Equal(t, 10*time.Second, runner.PollInterval)
		assert.Equal(t, 5, runner.WorkersCount)
		assert.DeepEqual(t, []bus.AsyncRequest{
			deployCommand,
		}, runner.Messages)

		runner = runners[1]
		assert.Equal(t, 10*time.Second, runner.PollInterval)
		assert.Equal(t, 3, runner.WorkersCount)
		assert.ArrayEqualFunc(t, []bus.AsyncRequest{
			someOtherMessage,
			anotherMessage,
		}, runner.Messages, compareRequestFunc)
	})

	t.Run("should correctly handle environment values as taking precedence", func(t *testing.T) {
		mapper := buildMapper(someOtherMessage, anotherMessage, deployCommand)
		logger, conf, err := loadConfiguration("config-to-override.yml", map[string]string{
			"LOG_LEVEL":               "info",
			"LOG_FORMAT":              "console",
			"DATA_PATH":               "testdata",
			"HTTP_HOST":               "192.0.1.68",
			"HTTP_PORT":               "8080",
			"HTTP_SECRET":             "my_secret",
			"ADMIN_EMAIL":             "admin@example.com",
			"ADMIN_PASSWORD":          "mypassword",
			"SOURCE_ARCHIVE_MAX_SIZE": "20mb",
			"EXPOSED_ON":              "https://seelf.somewhere.com",
			"RUNNERS_CONFIGURATION":   "4s;2;deployment.command.deploy|4s;3;",
		})

		assert.Nil(t, err)
		assert.Equal(t, log.InfoLevel, logger.level)
		assert.Equal(t, log.OutputConsole, logger.format)
		assert.False(t, conf.IsDebug())
		assert.True(t, conf.IsSecure())
		assert.Equal(t, "192.0.1.68:8080", conf.ListenAddress())
		assert.Equal(t, "file:testdata/seelf.db?_journal=WAL&_timeout=5000&_foreign_keys=yes&_txlock=immediate&_synchronous=NORMAL", conf.ConnectionString())
		assert.Equal(t, "testdata", conf.DataDir())
		assert.Equal(t, 20971520, conf.MaxDeploymentArchiveFileSize())
		assert.DeepEqual(t, []byte("my_secret"), conf.Secret())
		assert.Equal(t, "admin@example.com", conf.DefaultEmail())
		assert.Equal(t, "mypassword", conf.DefaultPassword())
		assert.True(t, conf.AppExposedUrl().HasValue())
		assert.Equal(t, "https://seelf.somewhere.com", conf.AppExposedUrl().MustGet().String())

		runners, err := conf.RunnersDefinitions(mapper)
		assert.Nil(t, err)
		assert.HasLength(t, 2, runners)

		runner := runners[0]
		assert.Equal(t, 4*time.Second, runner.PollInterval)
		assert.Equal(t, 2, runner.WorkersCount)
		assert.DeepEqual(t, []bus.AsyncRequest{
			deployCommand,
		}, runner.Messages)

		runner = runners[1]
		assert.Equal(t, 4*time.Second, runner.PollInterval)
		assert.Equal(t, 3, runner.WorkersCount)
		assert.ArrayEqualFunc(t, []bus.AsyncRequest{
			someOtherMessage,
			anotherMessage,
		}, runner.Messages, compareRequestFunc)
	})

	t.Run("should fail to build runners definition if some jobs are not handled", func(t *testing.T) {
		mapper := buildMapper(someOtherMessage, anotherMessage, deployCommand)
		_, config, err := loadConfiguration("valid-config.yml", map[string]string{
			"RUNNERS_CONFIGURATION": "4s;2;deployment.command.deploy,some.other.message",
		})

		assert.Nil(t, err)

		_, err = config.RunnersDefinitions(mapper)

		assert.NotNil(t, err)
		assert.Equal(t, "some background jobs are not handled: and.another.message, please fix your configuration by adding a worker to handle them", err.Error())
	})

	t.Run("should fail to build runners definition if some jobs does not exist", func(t *testing.T) {
		mapper := buildMapper(someOtherMessage, anotherMessage, deployCommand)
		_, config, err := loadConfiguration("valid-config.yml", map[string]string{
			"RUNNERS_CONFIGURATION": "4s;2;non.existent.job|4s;2;",
		})

		assert.Nil(t, err)

		_, err = config.RunnersDefinitions(mapper)

		assert.NotNil(t, err)
		assert.Match(t, "unknown job name: non.existent.job, must be one of .*,.*", err.Error())
	})
}

func loadConfiguration(configName string, envValues map[string]string) (*testLogger, config.Configuration, error) {
	confFilename := filepath.Join("testdata", configName)
	logger := &testLogger{
		ConfigurableLogger: must.Panic(log.NewLogger()),
	}

	os.Clearenv()

	for k, v := range envValues {
		os.Setenv(k, v)
	}

	conf := config.Default()
	err := conf.Initialize(logger, confFilename)

	return logger, conf, err
}

func buildMapper(messages ...message) *storage.DiscriminatedMapper[bus.AsyncRequest] {
	mapper := storage.NewDiscriminatedMapper(func(a bus.AsyncRequest) string { return a.Name_() })

	for _, m := range messages {
		mapper.Register(m, func(s string) (bus.AsyncRequest, error) { return m, nil })
	}

	return mapper
}

type testLogger struct {
	log.ConfigurableLogger
	format log.OutputFormat
	level  log.Level
}

func (t *testLogger) Configure(format log.OutputFormat, level log.Level) error {
	t.format = format
	t.level = level

	return t.ConfigurableLogger.Configure(format, level)
}

type message struct {
	bus.AsyncCommand

	name string
}

func (m message) Name_() string { return m.name }
func (m message) Group() string { return "" }

func compareRequestFunc(a, b bus.AsyncRequest) int { return strings.Compare(a.Name_(), b.Name_()) }
