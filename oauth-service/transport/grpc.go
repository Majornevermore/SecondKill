package transport

import (
	"SecondKill/oauth-service/endpoint"
	"SecondKill/pb"
	"context"
	"github.com/go-kit/kit/transport/grpc"
)

type grpcServer struct{
	checkTokenServer grpc.Handler
}

func (s *grpcServer) CheckToken(ctx context.Context, request *pb.CheckTokenRequest) (*pb.CheckTokenResponse, error) {
	_, resp, err := s.checkTokenServer.ServeGRPC(ctx, request)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.CheckTokenResponse), nil
}

func NewGRPCServer(ctx context.Context, endpoints endpoint.OAuth2Endpoints, serverTracer grpc.ServerOption) pb.OAuthServiceServer {
	return &grpcServer{
		checkTokenServer : grpc.NewServer(
			endpoints.GRPCCheckTokenEndpoint,
			DecodeGRPCCheckTokenRequest,
			EncodeGRPCCheckTokenResponse,
			serverTracer,
			),
	}
}


func DecodeGRPCCheckTokenRequest(ctx context.Context, r interface{}) (interface{}, error) {
	req := r.(*pb.CheckTokenRequest)
	return &endpoint.CheckTokenRequest{
		Token: req.Token,
	}, nil
}


func EncodeGRPCCheckTokenResponse(_ context.Context, r interface{}) (interface{}, error) {
	resp := r.(endpoint.CheckTokenResponse)

	if resp.Error != "" {
		return &pb.CheckTokenResponse{
			IsValidToken: false,
			Err:          resp.Error,
		}, nil
	} else {
		return &pb.CheckTokenResponse{
			UserDetails: &pb.UserDetails{
				UserId:      resp.OAuthDetails.User.UserId,
				Username:    resp.OAuthDetails.User.Username,
				Authorities: resp.OAuthDetails.User.Authorities,
			},
			ClientDetails: &pb.ClientDetails{
				ClientId:                    resp.OAuthDetails.Client.ClientId,
				AccessTokenValiditySeconds:  int32(resp.OAuthDetails.Client.AccessTokenValiditySeconds),
				RefreshTokenValiditySeconds: int32(resp.OAuthDetails.Client.RefreshTokenValiditySeconds),
				AuthorizedGrantTypes:        resp.OAuthDetails.Client.AuthorizedGrantTypes,
			},
			IsValidToken: true,
			Err:          "",
		}, nil
	}
}
