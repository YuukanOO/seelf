package config

import (
	"errors"
	"io/fs"
	"os"

	nenv "github.com/Netflix/go-env"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/joho/godotenv"

	"gopkg.in/yaml.v3"
)

var ErrNoLoadersGiven = errors.New("no loaders given")

type Loader func(any) error

// Load the configuration into the target using given loaders.
func Load(target any, loaders ...Loader) error {
	if len(loaders) == 0 {
		return ErrNoLoadersGiven
	}

	for _, loader := range loaders {
		if err := loader(target); err != nil {
			return err
		}
	}

	return nil
}

// Save the given config data in the given yaml file.
func Save(configFilePath string, data any) error {
	b, err := yaml.Marshal(data)

	if err != nil {
		return err
	}

	return ostools.WriteFile(configFilePath, b)
}

// Load the configuration from the given yaml file.
// Update the found argument to true if the file was found.
func FromYAML(path string, found *bool) Loader {
	return func(target any) error {
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				*found = false
				return nil
			}

			return err
		}

		*found = true

		data, err := os.ReadFile(path)

		if err != nil {
			return err
		}

		return yaml.Unmarshal(data, target)
	}
}

// Load the configuration from environment variables, trying to read given
// .env filenames before.
func FromEnvironment(dotEnvFilenames ...string) Loader {
	return func(target any) error {
		for _, filename := range dotEnvFilenames {
			if err := godotenv.Load(filename); err != nil && !os.IsNotExist(err) {
				return err
			}
		}

		_, err := nenv.UnmarshalFromEnviron(target)
		return err
	}
}
