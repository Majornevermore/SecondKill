package endpoint

import (
	"SpeedKill/oauth-service/model"
	"context"
	"errors"
	"github.com/go-kit/kit/endpoint"
	"log"
	"net/http"
)

const (
	OAuth2DetailsKey       = "OAuth2Details"
	OAuth2ClientDetailsKey = "OAuth2ClientDetails"
	OAuth2ErrorKey         = "OAuth2Error"
)

var (
	ErrInvalidClientRequest = errors.New("invalid client message")
	ErrInvalidUserRequest   = errors.New("invalid user message")
	ErrNotPermit            = errors.New("not permit")
)

func MakeClientAuthorizationMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if err, ok := ctx.Value(OAuth2ErrorKey).(error); ok {
				return nil, err
			}
			if _, ok := ctx.Value(OAuth2ClientDetailsKey).(model.ClientDetails); !ok {
				return nil, ErrInvalidClientRequest
			}
			return next(ctx, request)
		}
	}
}

type TokenRequest struct {
	GrantType string
	Reader    *http.Request
}

type TokenResponse struct {
	AccessToken *model.OAuth2Token `json:"access_token"`
	Error       string             `json:"error"`
}
