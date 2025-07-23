package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	HTTPServerHost       string `mapstructure:"HTTP_SERVER_HOST"`
	ProcessorDefaultURL  string `mapstructure:"PROCESSOR_DEFAULT_URL"`
	ProcessorFallbackURL string `mapstructure:"PROCESSOR_FALLBACK_URL"`
	PubsubURL            string `mapstructure:"PUBSUB_URL"`
	DatabaseURL          string `mapstructure:"DATABASE_URL"`
	DatabaseURLPostgres  string `mapstructure:"DATABASE_URL_POSTGRES"`
	MaxJobs              int    `mapstructure:"MAX_JOBS"`
	JobTimeout           int    `mapstructure:"JOB_TIMEOUT"`
	RedisURL             string `mapstructure:"REDIS_URL"`
	RedisPassword        string `mapstructure:"REDIS_PASSWORD"`

	RPCAddr string `mapstructure:"RPC_ADDR"`
	Debug   bool   `mapstructure:"DEBUG"`
}

func LoadConfig(path ...string) Config {
	viper.AddConfigPath("./etc/config/")
	viper.SetConfigName("server.env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("unable to decode into struct, %v", err))
	}

	return config
}
