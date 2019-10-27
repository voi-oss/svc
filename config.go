package svc

import (
	"github.com/caarlos0/env"
	"gopkg.in/go-playground/validator.v9"
)

// LoadFromEnv parses environment variables into a given struct and validates
// its fields' values.
func LoadFromEnv(config interface{}) error {
	if err := env.Parse(config); err != nil {
		return err
	}
	if err := validator.New().Struct(config); err != nil {
		return err
	}
	return nil
}
