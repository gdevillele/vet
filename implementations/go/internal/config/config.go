package config

const DefaultMaxFunctionParameters = 1

type Config struct {
	MaxFunctionParameters MaxFunctionParametersRule
}

type MaxFunctionParametersRule struct {
	Enabled bool
	Max     int
}

func Default() Config {
	return Config{
		MaxFunctionParameters: MaxFunctionParametersRule{
			Enabled: true,
			Max:     DefaultMaxFunctionParameters,
		},
	}
}
