package client

import (
	"SecondKill/pb"
	"SecondKill/pkg/discover"
	"SecondKill/pkg/loadbalance"
	"context"
	"github.com/opentracing/opentracing-go"
)

type UserClient interface {
	CheckUser(ctx context.Context, tracer opentracing.Tracer, requeset *pb.UserRequest) (*pb.UserResponse, error)
}

type UserClientImpl struct {
	manager     ClientManager
	serviceName string
	loadBalance loadbalance.Balance
	tracer      opentracing.Tracer
}

func (u *UserClientImpl) CheckUser(ctx context.Context, tracer opentracing.Tracer, requeset *pb.UserRequest) (*pb.UserResponse, error) {
	response := new(pb.UserResponse)
	if err := u.manager.DecoratorInvoke("pb.UserService/check", "user_check", tracer, ctx, requeset, response); err != nil {
		return nil, err
	} else {
		return response, nil
	}
}

func NewUserClient(serviceName string, lb loadbalance.Balance, tracer opentracing.Tracer) (UserClient, error) {
	if serviceName == "" {
		serviceName = "user"
	}
	if lb == nil {
		lb = defaultLoadBalance
	}

	return &UserClientImpl{
		manager: &DefaultClientManager{
			serviceName:    serviceName,
			loadbalance:    lb,
			discoverClient: discover.ConsulService,
			logger:         discover.Logger,
		},
		serviceName: serviceName,
		loadBalance: lb,
		tracer:      tracer,
	}, nil
}
