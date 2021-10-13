package svc

import (
	"reflect"

	"github.com/caarlos0/env/v6"
	"github.com/go-playground/validator/v10"
)

// LoadFromEnv parses environment variables into a given struct and validates
// its fields' values.
func LoadFromEnv(config interface{}) error {
	return LoadFromEnvWithParsers(config, nil)
}

// LoadFromEnvWithParsers parses environment variables into a given struct and validates
// its fields' values, also allows for custom type parsers
func LoadFromEnvWithParsers(config interface{}, parsers map[reflect.Type]env.ParserFunc) error {
	if err := env.ParseWithFuncs(config, parsers); err != nil {
		return err
	}
	if err := validator.New().Struct(config); err != nil {
		return err
	}
	return nil
}
