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

// Load the configuration into the target from a yaml file and environment variables.
//
// The boolean returned is true if the config file has been found, false otherwise.
//
// It will look for dotenv files in the current directory. If no dotenvFiles are given,
// default ones will be used: .env and .env.local.
// target can implement the Processable interface to do any stuff after the config has been loaded.
func Load(configFilePath string, target any, dotenvFiles ...string) (exists bool, err error) {
	if exists, err = loadFromYaml(configFilePath, target); err != nil {
		return
	}

	if len(dotenvFiles) == 0 {
		dotenvFiles = dotenvFilenames
	}

	err = loadFromEnvironment(dotenvFiles, target)

	return
}

// Save the given config data in the given file path.
func Save(configFilePath string, data any) error {
	b, err := yaml.Marshal(data)

	if err != nil {
		return err
	}

	return ostools.WriteFile(configFilePath, b)
}

func loadFromYaml(path string, target any) (bool, error) {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	data, err := os.ReadFile(path)

	if err != nil {
		return true, err
	}

	return true, yaml.Unmarshal(data, target)
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
