package proxy

import (
	"fmt"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	BaseUrl  string
	HttpPort string
	GrpcPort string
	UseTls   bool
}

func ReadConfig() (*ServerConfig, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	err := v.ReadInConfig()
	v.AutomaticEnv()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Warning: config file does not exist")
		} else {
			return nil, err
		}
	}
	return &ServerConfig{
		BaseUrl:  v.GetString("BASE_URL"),
		HttpPort: v.GetString("HTTP_PORT"),
		GrpcPort: v.GetString("GRPC_PORT"),
		UseTls:   v.GetBool("USE_TLS"),
	}, nil
}
