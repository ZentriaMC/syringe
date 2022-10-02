package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

func LoadConfig(filename string) (c *Config, err error) {
	var file *os.File
	if file, err = os.OpenFile(filename, os.O_RDONLY, 0); err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	var config Config
	if err = yaml.NewDecoder(file).Decode(&config); err != nil {
		return
	}

	return &config, nil
}

type Config struct {
	Templates []Template `yaml:"templates"`
}

type Template struct {
	Unit       string        `yaml:"unit"`
	Credential []string      `yaml:"credential"`
	Contents   string        `yaml:"contents"`
	Options    *TemplateOpts `yaml:"options"`
}

type TemplateOpts struct {
	DelimLeft    string `yaml:"delim_left"`
	DelimRight   string `yaml:"delim_right"`
	SandboxPath  string `yaml:"sandbox_path"`
	AllowMissing bool   `yaml:"allow_missing"`
}
