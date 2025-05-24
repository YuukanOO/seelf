package config

import (
	"errors"
	"strconv"
	"strings"

	"github.com/YuukanOO/seelf/internal/deployment/app/deploy"
	"gopkg.in/yaml.v3"
)

const (
	runnersEnvExpectedParts     = 3
	runnersEnvPartSeparator     = ";"
	runnersEnvJobNamesSeparator = ","
	runnersEnvSeparator         = "|"
)

type (
	runnersConfiguration []runnerConfiguration

	deprecatedRunnerConfiguration struct {
		PollInterval string `yaml:"poll_interval"`
		Deployment   int    `yaml:"deployment"`
		Cleanup      int    `yaml:"cleanup"`
	}
)

var ErrRunnersEnvParseFailed = errors.New("failed to parse runners configuration from environment")

func (r *runnersConfiguration) UnmarshalYAML(value *yaml.Node) error {
	initialErr := r.tryDecodeCurrentFormat(value)

	if initialErr == nil {
		return nil
	}

	if err := r.tryDecodeDeprecatedFormat(value); err != nil {
		// If there's still an error, just returns the initial one as it's probably the one we want
		return initialErr
	}

	return nil
}

func (r *runnersConfiguration) UnmarshalEnvironmentValue(data string) error {
	// No need to go further if the value is empty
	if data == "" {
		return ErrRunnersEnvParseFailed
	}

	runners := strings.Split(data, runnersEnvSeparator)

	*r = make(runnersConfiguration, len(runners))

	for i, runnerStr := range runners {
		parts := strings.SplitN(runnerStr, runnersEnvPartSeparator, runnersEnvExpectedParts)

		if len(parts) != runnersEnvExpectedParts {
			return ErrRunnersEnvParseFailed
		}

		conf := runnerConfiguration{
			PollInterval: parts[0],
		}

		if parts[2] != "" {
			conf.Jobs = strings.Split(parts[2], runnersEnvJobNamesSeparator)
		}

		workersCount, err := strconv.Atoi(parts[1])

		if err != nil {
			return ErrRunnersEnvParseFailed
		}

		conf.Count = uint8(workersCount)

		(*r)[i] = conf
	}

	return nil
}

func (r *runnersConfiguration) tryDecodeCurrentFormat(value *yaml.Node) error {
	var runnersConf []runnerConfiguration

	if err := value.Decode(&runnersConf); err != nil {
		return err
	}

	*r = runnersConf
	return nil
}

func (r *runnersConfiguration) tryDecodeDeprecatedFormat(value *yaml.Node) error {
	var oldRunnersConfiguration deprecatedRunnerConfiguration

	if err := value.Decode(&oldRunnersConfiguration); err != nil {
		return err
	}

	// Migrate to the new configuration
	*r = defaultRunnersConfiguration(
		oldRunnersConfiguration.PollInterval,
		uint8(oldRunnersConfiguration.Deployment),
		uint8(oldRunnersConfiguration.Cleanup),
	)
	return nil
}

func defaultRunnersConfiguration(pollInterval string, deploymentsCount uint8, generalCount uint8) runnersConfiguration {
	return runnersConfiguration{
		{
			PollInterval: pollInterval,
			Count:        deploymentsCount,
			Jobs:         []string{deploy.Command{}.Name_()},
		},
		{
			PollInterval: pollInterval,
			Count:        generalCount,
		},
	}
}
