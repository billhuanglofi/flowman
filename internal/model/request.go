package model

type Request struct {
	Version        string          `yaml:"version"`
	Name           string          `yaml:"name"`
	Method         string          `yaml:"method"`
	URL            string          `yaml:"url,omitempty"`
	Endpoint       string          `yaml:"endpoint,omitempty"`
	Path           string          `yaml:"path,omitempty"`
	Query          []Parameter     `yaml:"query,omitempty"`
	Headers        []Header        `yaml:"headers,omitempty"`
	Body           RequestBody     `yaml:"body,omitempty"`
	Trace          TraceConfig     `yaml:"trace,omitempty"`
	ImportWarnings []ImportWarning `yaml:"import_warnings,omitempty"`
}

type Parameter struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Header struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value,omitempty"`
	Env   string `yaml:"env,omitempty"`
}

type RequestBody struct {
	Mode string      `yaml:"mode,omitempty"`
	Raw  string      `yaml:"raw,omitempty"`
	Form []Parameter `yaml:"form,omitempty"`
}
