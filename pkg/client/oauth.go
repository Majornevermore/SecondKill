package client

import (
	"SpeedKill/pb"
	"SpeedKill/pkg/discover"
	"SpeedKill/pkg/loadbalance"
	"context"
	"github.com/opentracing/opentracing-go"
)

type OAuthClient interface {
	CheckToken(ctx context.Context, tracer opentracing.Tracer, request *pb.CheckTokenRequest ) (*pb.CheckTokenResponse, error)
}

func (O *OAuthClientImpl) CheckToken(ctx context.Context, tracer opentracing.Tracer, request *pb.CheckTokenRequest) (*pb.CheckTokenResponse, error) {
	response := new(pb.CheckTokenResponse)
	if err := O.manager.DecoratorInvoke("pb.OAuthService/CheckToken", "token_check", tracer, ctx, request, response); err != nil {
		return nil, err
	} else {
		return response, nil
	}
}

type OAuthClientImpl struct {
	manager     ClientManager
	serviceName string
	loadBalance loadbalance.Balance
	tracer      opentracing.Tracer
}

func NewOAuthClient(serviceName string, lb loadbalance.Balance, tracer opentracing.Tracer) (OAuthClient, error) {
	if serviceName == "" {
		serviceName = "oauth"
	}
	if lb == nil {
		lb = defaultLoadBalance
	}

	return &OAuthClientImpl{
		manager: &DefaultClientManager{
			serviceName: serviceName,
			loadbalance: lb,
			discoverClient:discover.ConsulService,
			logger:discover.Logger,
		},
		serviceName: serviceName,
		loadBalance: lb,
		tracer:      tracer,
	}, nil
}
