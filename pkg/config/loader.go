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

var dotenvFilenames = []string{".env", ".env.local"}

// Processable is an interface that can be implemented by the target of the Load function to
// do any stuff after a config has been loaded.
type Processable interface {
	PostLoad() error
}

// Load the configuration into the target from a yaml file and environment variables.
// It will look for dotenv files in the current directory.
// target can implement the Processable interface to do any stuff after the config has been loaded.
func Load(configFilePath string, target any) error {
	if err := loadFromYaml(configFilePath, target); err != nil {
		return err
	}

	if err := loadFromEnvironment(dotenvFilenames, target); err != nil {
		return err
	}

	postProcessable, ok := target.(Processable)

	if !ok {
		return nil
	}

	return postProcessable.PostLoad()
}

// Save the given config data in the given file path.
func Save(configFilePath string, data any) error {
	b, err := yaml.Marshal(data)

	if err != nil {
		return err
	}

	return ostools.WriteFile(configFilePath, b)
}

func loadFromYaml(path string, target any) error {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	data, err := os.ReadFile(path)

	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, target)
}

func loadFromEnvironment(filenames []string, target any) error {
	for _, filename := range filenames {
		if err := godotenv.Load(filename); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	_, err := nenv.UnmarshalFromEnviron(target)
	return err
}
