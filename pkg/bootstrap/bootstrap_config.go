package bootstrap

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
)

func init() {
	viper.AutomaticEnv()
	initBootstrapConfig()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err)
	}
	if err := subParse("http", &HttpConfig); err != nil {
		fmt.Printf("parse http err")
	}
	if err := subParse("discover", &DiscoverConfig); err != nil {
		log.Fatal("Fail to parse Discover config", err)
	}
	if err := subParse("config", &ConfigServerConfig); err != nil {
		log.Fatal("Fail to parse config server", err)
	}
	if err := subParse("rpc", &RpcConfig); err != nil {
		log.Fatal("Fail to parse rpc server", err)
	}
}

func initBootstrapConfig()  {
	viper.SetConfigName("bootstrap")
	viper.AddConfigPath("./")
	viper.AddConfigPath("$GOPATH/src/")
	viper.SetConfigType("yaml")
}

func subParse(key string, value interface{})  error {
	log.Printf("配置前缀为:%v", key)
	sub :=viper.Sub(key)
	sub.AutomaticEnv()
	sub.SetEnvPrefix(key)
	return sub.Unmarshal(value)
}