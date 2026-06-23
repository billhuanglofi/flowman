package model

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

type TraceConfig struct {
	TransactionID TransactionIDExtraction `yaml:"transaction_id"`
	PollInterval  Duration                `yaml:"poll_interval"`
	Timeout       Duration                `yaml:"timeout"`
}

type TransactionIDExtraction struct {
	Header    string   `yaml:"header,omitempty"`
	JSONPaths []string `yaml:"json_paths,omitempty"`
	Regex     string   `yaml:"regex,omitempty"`
}

type Duration time.Duration

func NewDuration(value time.Duration) Duration {
	return Duration(value)
}

func (duration Duration) Duration() time.Duration {
	return time.Duration(duration)
}

func (duration Duration) String() string {
	return time.Duration(duration).String()
}

func (duration Duration) MarshalYAML() (any, error) {
	return duration.String(), nil
}

func (duration *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("trace duration must be a scalar")
	}
	parsed, err := time.ParseDuration(value.Value)
	if err != nil {
		return fmt.Errorf("trace duration %q: %w", value.Value, err)
	}
	*duration = Duration(parsed)
	return nil
}
