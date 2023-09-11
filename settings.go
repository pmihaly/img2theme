package main

import (
	"gopkg.in/yaml.v2"
	"os"
)

type Settings struct {
	Palette         []ColorfulColor `yaml:"palette"`
	PaletteAffinity float64         `yaml:"palette-affinity"`
	Cpus            int             `yaml:"cpus"`
}

func loadSettingsFromYaml(filePath string) (Settings, error) {
	rawSettings, err := os.ReadFile(filePath)
	if err != nil {
		return Settings{}, err
	}

	settings := Settings{}
	err = yaml.Unmarshal(rawSettings, &settings)
	if err != nil {
		return Settings{}, err
	}

	return settings, nil
}
