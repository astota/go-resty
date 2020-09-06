package rest

import (
	"github.com/astota/go-logging"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

// Configuration to handle server properties
type Configuration struct {
	// Application name identifies which application is in use
	ApplicationName string `yaml:"application" json:"application"`
	// Maximum duration that handling of requst can take. Default : 30s
	MaximumRequestDuration time.Duration `yaml:"max_request_duration" json:"max_request_duration"`
	// Maximum request body size. Default: 1MB
	MaximumBodySize int64 `yaml:"max_body_size" json:"max_body_size"`
	// Log Level. Default: info
	LogLevel string `yaml:"log_level" json:"log_level"`
	// Shutdown grace time, time which is waited before force shutdown.
	// Defafult 30s
	ShutdownGraceTime time.Duration `yaml:"shutdown_grace_time" json:"shutdown_grace_time"`
}

// NewConfiguration Creates new middleware configuration. Default values are
//
// - MaximumRequestDuration: 30s
//
// - MaximumBodySize: 1MB
//
// - LogLevel: info
func NewConfiguration() Configuration {
	return Configuration{
		ApplicationName:        "test_app",
		MaximumRequestDuration: 30 * time.Second,
		MaximumBodySize:        1 << 20,
		LogLevel:               "info",
		ShutdownGraceTime:      30 * time.Second,
	}
}

// ReadConfiguration reads configuration YAML file and parses
// configuration struct. If there is error in parsing, error
// is returned and configuration is unchanged.
func ReadConfiguration(fn string, conf interface{}) error {
	logger := logging.NewLogger()

	d, err := ioutil.ReadFile(fn)
	if err != nil {
		logger.Error(fmt.Sprintf("Error when reading config file '%s': %s", fn, err.Error()))
		return err
	} else if d != nil {
		if err := yaml.Unmarshal(d, conf); err != nil {
			logger.Error(fmt.Sprintf("configuration invalid: %s", err.Error()))
			return err
		}
	} else {
		logger.Error(fmt.Sprintf("No data found in config file '%s'", fn))
		return errors.New("No data found in config file")
	}

	return nil
}
