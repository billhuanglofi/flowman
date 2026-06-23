package model

type ImportWarning struct {
	Code    string `yaml:"code"`
	Path    string `yaml:"path"`
	Message string `yaml:"message"`
}
