package config_test

import (
	"os"
	"path/filepath"
	"slices"
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
		logger, conf, err := loadConfiguration("valid-config.yml")

		assert.Nil(t, err)
		assert.Equal(t, log.DebugLevel, logger.level)
		assert.Equal(t, log.OutputJSON, logger.format)
		assert.True(t, conf.IsDebug())
		assert.True(t, conf.IsSecure())
		assert.Equal(t, "localhost:5000", conf.ListenAddress())
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
		assert.DeepEqual(t, sortMessages([]bus.AsyncRequest{
			someOtherMessage,
			anotherMessage,
		}), sortMessages(runner.Messages))
	})

	t.Run("should correctly handle deprecated runners configuration", func(t *testing.T) {
		mapper := buildMapper(someOtherMessage, anotherMessage, deployCommand)
		_, conf, err := loadConfiguration("deprecated-runners-config.yml")

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
		// TODO: maybe add an assert.ArrayEqual[T any](t, []T, []T, comparer func(T,T) int)
		assert.DeepEqual(t, sortMessages([]bus.AsyncRequest{
			someOtherMessage,
			anotherMessage,
		}), sortMessages(runner.Messages))
	})

	t.Run("should correctly handle environment values as taking precedence", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("should fail to build runners definition if some jobs are not handled", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("should fail to build runners definition if some jobs does not exist", func(t *testing.T) {
		t.Skip("TODO")
	})
}

func loadConfiguration(configName string) (*testLogger, config.Configuration, error) {
	confFilename := filepath.Join("testdata", configName)
	logger := &testLogger{
		ConfigurableLogger: must.Panic(log.NewLogger()),
	}

	os.Clearenv()

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

func sortMessages(requests []bus.AsyncRequest) []bus.AsyncRequest {
	slices.SortFunc(requests, func(a, b bus.AsyncRequest) int { return strings.Compare(a.Name_(), b.Name_()) })
	return requests
}
