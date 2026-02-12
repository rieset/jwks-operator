package config

import (
	"fmt"
	"time"
)

// Duration is a custom type for parsing duration from YAML
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("failed to parse duration '%s': %w", s, err)
	}

	d.Duration = duration
	return nil
}

// MarshalYAML implements yaml.Marshaler interface
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}
