package main

import (
	localconfig "SpeedKill/oauth-service/config"
	"SpeedKill/oauth-service/endpoint"
	"SpeedKill/oauth-service/plugins"
	"SpeedKill/oauth-service/service"
	"SpeedKill/pkg/bootstrap"
	"SpeedKill/pkg/config"
	"SpeedKill/pkg/mysql"
	"flag"
	"fmt"
	kitzipkin "github.com/go-kit/kit/tracing/zipkin"
	"golang.org/x/time/rate"
	"time"
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
		srv                  service.Service
	)
	ratebucket := rate.NewLimiter(rate.Every(time.Second*1), 100)
	srv = service.NewCommentService()
	tokenEnhancer = service.NewJwtTokenEnhancer("secret")
	tokenStore = service.NewJwtTokenStore(tokenEnhancer.(*service.JwtTokenEnhancer))
	tokenService = service.NewTokenService(tokenStore, tokenEnhancer)
	userDetailsService = service.NewRemoteUserDetailService()
	clientDetailsService = service.NewMysqlClientDetailsService()
	passWordGranter := service.NewUsernamePasswordTokenGranter("password", userDetailsService, tokenService)
	refreshGranter := service.NewRefreshGranter("refresh_token", userDetailsService, tokenService)
	tokenGranter = service.NewComposeTokenGrante(map[string]service.TokenGranter{
		"password":      passWordGranter,
		"refresh_token": refreshGranter,
	})
	tokenEndpoint := endpoint.MakeTokenEndPoint(tokenGranter, clientDetailsService)
	tokenEndpoint = endpoint.MakeClientAuthorizationMiddleware(localconfig.Logger)(tokenEndpoint)
	tokenEndpoint = plugins.NewTokenBucketLimitterWithBuildIn(ratebucket)(tokenEndpoint)
	tokenEndpoint = kitzipkin.TraceEndpoint(localconfig.ZipkinTracer, "token-endpoint")(tokenEndpoint)

	checkEndpoint := endpoint.MakeCheckTokenEndpoint(tokenService)
	checkEndpoint = endpoint.MakeClientAuthorizationMiddleware(localconfig.Logger)(checkEndpoint)
	checkEndpoint = plugins.NewTokenBucketLimitterWithBuildIn(ratebucket)(checkEndpoint)
	checkEndpoint = kitzipkin.TraceEndpoint(localconfig.ZipkinTracer, "check-endpoint")(checkEndpoint)

	gRPCCheckTokenEndpoint := endpoint.MakeCheckTokenEndpoint(tokenService)
	gRPCCheckTokenEndpoint = plugins.NewTokenBucketLimitterWithBuildIn(ratebucket)(gRPCCheckTokenEndpoint)
	gRPCCheckTokenEndpoint = kitzipkin.TraceEndpoint(localconfig.ZipkinTracer, "grpc-check-endpoint")(gRPCCheckTokenEndpoint)

	//创建健康检查的Endpoint
	healthEndpoint := endpoint.MakeHealthCheckEndpoint(srv)
	healthEndpoint = kitzipkin.TraceEndpoint(localconfig.ZipkinTracer, "health-endpoint")(healthEndpoint)
	endpts := endpoint.OAuth2Endpoints{
		TokenEndpoint:          tokenEndpoint,
		CheckTokenEndpoint:     checkTokenEndpoint,
		HealthCheckEndpoint:    healthEndpoint,
		GRPCCheckTokenEndpoint: gRPCCheckTokenEndpoint,
	}

	//创建http.Handler
	r := transport.MakeHttpHandler(ctx, endpts, tokenService, clientDetailsService, localconfig.ZipkinTracer, localconfig.Logger)

	// http server
	go func() {
		fmt.Printf("http server start at port:" + *servicePort)
		mysql.InitMysql(config.MysqlConfig.Host, config.MysqlConfig.Port, config.MysqlConfig.User,
			config.MysqlConfig.Pwd, config.MysqlConfig.Db)
	}()
}
