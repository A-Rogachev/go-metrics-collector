package config

import (
	"flag"
	"os"
)

func ConfigParamStringCounter(envName string, flagName string) *string {
	envValue, ok := os.LookupEnv(envName)
	if ok {
		return &envValue
	}

	var value *string
	flag.Visit(func(f *flag.Flag) {
		if f.Name == flagName {
			flagValue := f.Value.String()
			value = &flagValue
		}
	})
	return value
}
