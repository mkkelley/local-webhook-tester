package proxy

import (
	"github.com/spf13/viper"
)

type ServerConfig struct {
	BaseUrl  string
	HttpPort string
	GrpcPort string
}

func readConfig() (*ServerConfig, error) {
	config := ServerConfig{}
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	err = v.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
