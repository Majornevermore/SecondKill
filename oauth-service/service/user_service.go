package service

import (
	"SecondKill/oauth-service/model"
	"SecondKill/pb"
	"SecondKill/pkg/client"
	"context"
	"errors"
)

type UserDetailsService interface {
	GetUserDetailByUserName(ctx context.Context, username, password string) (*model.UserDetails, error)
}

var (
	ErrUserNotExit  = errors.New("username is not exist")
	ErrPassword     = errors.New("password is err")
	InvalidUserInfo = errors.New("invalid user info")
)

type InMermoryUserDetailsService struct {
	userDetailsDict map[string]*model.UserDetails
}

func (service *InMermoryUserDetailsService) GetUserDetailByUserName(ctx context.Context, username, password string) (*model.UserDetails, error) {
	if userDitails, ok := service.userDetailsDict[username]; ok {
		if userDitails.Password == password {
			return userDitails, nil
		} else {
			return nil, ErrPassword
		}
	} else {
		return nil, ErrUserNotExit
	}
}

func NewInMermoryUserDetailsService(userDetailsList []*model.UserDetails) *InMermoryUserDetailsService {
	userDetailDic := make(map[string]*model.UserDetails)
	if userDetailsList != nil {
		for _, value := range userDetailsList {
			userDetailDic[value.Username] = value
		}
	}
	return &InMermoryUserDetailsService{
		userDetailsDict: userDetailDic,
	}
}

type RemoteUserService struct {
	userClient client.UserClient
}

func (service *RemoteUserService) GetUserDetailByUserName(ctx context.Context, username, password string) (*model.UserDetails, error) {
	response, err := service.userClient.CheckUser(ctx, nil, &pb.UserRequest{
		Username: username,
		Password: password,
	})
	if err == nil {
		if response.UserId != 0 {
			return &model.UserDetails{
				UserId:   response.UserId,
				Username: username,
				Password: password,
			}, nil
		} else {
			return nil, InvalidUserInfo
		}
	}
	return nil, err
}

func NewRemoteUserDetailService() *RemoteUserService {

	userClient, _ := client.NewUserClient("user", nil, nil)
	return &RemoteUserService{
		userClient: userClient,
	}
}
