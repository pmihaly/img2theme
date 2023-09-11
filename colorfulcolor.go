package main

import (
	"github.com/lucasb-eyer/go-colorful"
)

type ColorfulColor struct {
	colorful.Color
}

func (c *ColorfulColor) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var hex string
	if err := unmarshal(&hex); err != nil {
		return err
	}
	color, err := colorful.Hex(hex)
	if err != nil {
		return err
	}
	*c = ColorfulColor{color}
	return nil
}
