package main

import (
	"SpeedKill/oauth-service/service"
	"SpeedKill/pkg/bootstrap"
	"SpeedKill/pkg/config"
	"SpeedKill/pkg/mysql"
	"flag"
	"fmt"
)

func main() {
	var (
		servicePort = flag.String("service.port", bootstrap.HttpConfig.Port, "service port")
		grpcAddr    = flag.String("grpc", bootstrap.RpcConfig.Port, "gRPC listen address.")
	)

	var (
		tokenService         service.TokenService
		tokenGranter         service.TokenGranter
		tokenEnhancer        service.TokenEnhancer
		tokenStore           service.TokenStore
		userDetailsService   service.UserDetailsService
		clientDetailsService service.ClientDetailsService
	)
	tokenEnhancer = service.NewJwtTokenEnhancer("secret")
	tokenStore = service.NewJwtTokenStore(tokenEnhancer.(*service.JwtTokenEnhancer))
	tokenService = service.NewTokenService(tokenStore, tokenEnhancer)
	userDetailsService = service.NewRemoteUserDetailService()
	passWordGranter := service.NewUsernamePasswordTokenGranter("password", userDetailsService, tokenService)
	refreshGranter := service.NewRefreshGranter("refresh_token", userDetailsService, tokenService)
	tokenGranter = service.NewComposeTokenGrante(map[string]service.TokenGranter{
		"password":      passWordGranter,
		"refresh_token": refreshGranter,
	})
	tokenEndpoint := endpoint.MakeTokenEndpoint(tokenGranter, clientDetailsService)

	// http server
	go func() {
		fmt.Printf("http server start at port:" + *servicePort)
		mysql.InitMysql(config.MysqlConfig.Host, config.MysqlConfig.Port, config.MysqlConfig.User,
			config.MysqlConfig.Pwd, config.MysqlConfig.Db)
	}()
}
