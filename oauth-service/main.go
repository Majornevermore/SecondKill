package main

import (
	localconfig "SecondKill/oauth-service/config"
	"SecondKill/oauth-service/endpoint"
	"SecondKill/oauth-service/plugins"
	"SecondKill/oauth-service/service"
	"SecondKill/oauth-service/transport"
	"SecondKill/pb"
	"SecondKill/pkg/bootstrap"
	"SecondKill/pkg/config"
	register "SecondKill/pkg/discover"
	"SecondKill/pkg/mysql"
	"context"
	"flag"
	"fmt"
	kitzipkin "github.com/go-kit/kit/tracing/zipkin"
	"google.golang.org/grpc"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"golang.org/x/time/rate"
	"google.golang.org/grpc/metadata"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
		CheckTokenEndpoint:     checkEndpoint,
		HealthCheckEndpoint:    healthEndpoint,
		GRPCCheckTokenEndpoint: gRPCCheckTokenEndpoint,
	}
	ctx := context.Background()
	errChan := make(chan error)
	//创建http.Handler
	r := transport.MakeHttpHandler(ctx, endpts, tokenService, clientDetailsService, localconfig.ZipkinTracer, localconfig.Logger)

	// http server
	go func() {
		fmt.Printf("http server start at port:" + *servicePort)
		mysql.InitMysql(config.MysqlConfig.Host, config.MysqlConfig.Port, config.MysqlConfig.User,
			config.MysqlConfig.Pwd, config.MysqlConfig.Db)
		register.Register()
		handler := r
		errChan <- http.ListenAndServe(":"+*servicePort, handler)
	}()

	// grpc
	go func() {
		fmt.Println("grpc Server start at port:" + *grpcAddr)
		listener, err := net.Listen("tcp", ":"+*grpcAddr)
		if err != nil {
			errChan <- err
			return
		}
		serverTracer := kitzipkin.GRPCServerTrace(localconfig.ZipkinTracer, kitzipkin.Name("grpc-transport"))
		tr := localconfig.ZipkinTracer
		md := metadata.MD{}
		parentSpan := tr.StartSpan("test")
		b3.InjectGRPC(&md)(parentSpan.Context())
		ctx := metadata.NewIncomingContext(context.Background(), md)
		handler := transport.NewGRPCServer(ctx, endpts, serverTracer)
		gRPCServer := grpc.NewServer()
		pb.RegisterOAuthServiceServer(gRPCServer, handler)
		errChan <- gRPCServer.Serve(listener)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()
	error := <-errChan
	//服务退出取消注册
	register.DeRegister()
	fmt.Println(error)
}
