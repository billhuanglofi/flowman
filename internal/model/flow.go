package model

type Flow struct {
	Version        string          `yaml:"version"`
	Name           string          `yaml:"name"`
	Description    string          `yaml:"description,omitempty"`
	Steps          []FlowStep      `yaml:"steps"`
	ImportWarnings []ImportWarning `yaml:"import_warnings,omitempty"`
}

type FlowStep struct {
	Name    string `yaml:"name"`
	Request string `yaml:"request"`
	Trace   bool   `yaml:"trace,omitempty"`
}
