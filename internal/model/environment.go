package model

type Project struct {
	Version      string   `yaml:"version"`
	Name         string   `yaml:"name"`
	DefaultEnv   string   `yaml:"default_env"`
	Environments []string `yaml:"environments"`
	Requests     []string `yaml:"requests"`
	Flows        []string `yaml:"flows"`
}

type Environment struct {
	Version   string          `yaml:"version"`
	Name      string          `yaml:"name"`
	BaseURL   string          `yaml:"base_url"`
	Endpoints []EndpointAlias `yaml:"endpoints,omitempty"`
	Oracle    OracleDatabase  `yaml:"oracle"`
	Trace     TraceConfig     `yaml:"trace"`
}

type EndpointAlias struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type OracleDatabase struct {
	DSNEnv      string `yaml:"dsn_env"`
	UserEnv     string `yaml:"user_env"`
	PasswordEnv string `yaml:"password_env"`
}

type Workspace struct {
	Project      Project
	Environments []Environment
	Requests     []Request
	Flows        []Flow
}
