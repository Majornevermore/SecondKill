package service

import (
	"SpeedKill/oauth-service/model"
	"context"
	"errors"
	"github.com/dgrijalva/jwt-go"
	uuid "github.com/satori/go.uuid"
	"net/http"
	"strconv"
	"time"
)

var (
	ErrNotSupportGrantType               = errors.New("grant type is not supported")
	ErrNotSupportOperation               = errors.New("no support operation")
	ErrInvalidUsernameAndPasswordRequest = errors.New("invalid username, password")
	ErrInvalidTokenRequest               = errors.New("invalid token")
	ErrExpiredToken                      = errors.New("token is expired")
)

type TokenGranter interface {
	Grant(ctx context.Context, grantType string, client *model.ClientDetails, reader *http.Request) (*model.OAuth2Token, error)
}

type ComposeTokenGranter struct {
	TokenGrantDict map[string]TokenGranter
}

func (tokenGranter *ComposeTokenGranter) Grant(ctx context.Context, grantType string, client *model.ClientDetails, reader *http.Request) (*model.OAuth2Token, error) {
	dispathcGranter := tokenGranter.TokenGrantDict[grantType]
	if dispathcGranter == nil {
		return nil, ErrNotSupportGrantType
	}
	return dispathcGranter.Grant(ctx, grantType, client, reader)
}

func NewComposeTokenGrante(tokenGrantDict map[string]TokenGranter) TokenGranter {
	return &ComposeTokenGranter{
		TokenGrantDict: tokenGrantDict,
	}
}

type UsernamePasswordTokenGranter struct {
	supportGrantType   string
	userDetailsService UserDetailsService
	tokenService       TokenService
}

func (tokenGranter *UsernamePasswordTokenGranter) Grant(ctx context.Context, grantType string, client *model.ClientDetails, reader *http.Request) (*model.OAuth2Token, error) {
	if grantType != tokenGranter.supportGrantType {
		return nil, ErrNotSupportGrantType
	}
	// 从请求体中获取用户名密码
	username := reader.FormValue("username")
	password := reader.FormValue("password")
	if username == "" || password == "" {
		return nil, ErrInvalidUsernameAndPasswordRequest
	}
	// 验证用户名密码是否正确
	userDetails, err := tokenGranter.userDetailsService.GetUserDetailByUserName(ctx, username, password)
	if err != nil {
		return nil, ErrInvalidUsernameAndPasswordRequest
	}
	// 根据用户信息和客户端信息生成访问令牌
	return tokenGranter.tokenService.CreateAccessToken(&model.OAuth2Details{
		Client: client,
		User:   userDetails,
	})
}

func NewUsernamePasswordTokenGranter(grantType string, userDetailsService UserDetailsService, toekenService TokenService) TokenGranter {
	return &UsernamePasswordTokenGranter{
		supportGrantType:   grantType,
		userDetailsService: userDetailsService,
		tokenService:       toekenService,
	}
}

type RefreshTokenGranter struct {
	supportGranteType string
	tokenService      TokenService
}

func NewRefreshGranter(grantType string, userDetailsService UserDetailsService, tokenService TokenService) TokenGranter {
	return &RefreshTokenGranter{
		supportGranteType: grantType,
		tokenService:      tokenService,
	}
}

func (tokenGranter *RefreshTokenGranter) Grant(ctx context.Context, grantType string, client *model.ClientDetails, reader *http.Request) (*model.OAuth2Token, error) {
	if grantType != tokenGranter.supportGranteType {
		return nil, ErrNotSupportGrantType
	}
	// 从请求中获取刷新令牌
	refreshTokenValue := reader.URL.Query().Get("refresh_token")

	if refreshTokenValue == "" {
		return nil, ErrInvalidTokenRequest
	}

	return tokenGranter.tokenService.RefreshAccessToken(refreshTokenValue)

}

type TokenService interface {
	// 根据访问令牌获取对应的用户信息和客户端信息
	GetOAuth2DetailsByAccessToken(tokenValue string) (*model.OAuth2Token, error)
	// 根据用户信息和客户端信息生成访问令牌
	CreateAccessToken(oauth2Details *model.OAuth2Details) (*model.OAuth2Token, error)
	// 根据刷新令牌获取访问令牌
	RefreshAccessToken(refreshTokenValue string) (*model.OAuth2Token, error)
	// 根据用户信息和客户端信息获取已生成访问令牌
	GetAccessToken(details *model.OAuth2Details) (*model.OAuth2Token, error)
	// 根据访问令牌值获取访问令牌结构体
	ReadAccessToken(tokenValue string) (*model.OAuth2Token, error)
}

type DefaultTokenService struct {
	tokenStore    TokenStore
	tokenEnhancer TokenEnhancer
}

func NewTokenService(tokenStore TokenStore, tokenEnhancer TokenEnhancer) TokenService {
	return &DefaultTokenService{
		tokenStore:    tokenStore,
		tokenEnhancer: tokenEnhancer,
	}
}

func (tokenService *DefaultTokenService) GetOAuth2DetailsByAccessToken(tokenValue string) (*model.OAuth2Token, error) {
	return tokenService.tokenStore.ReadAccessToken(tokenValue)
}

func (tokenService *DefaultTokenService) CreateAccessToken(oauth2Details *model.OAuth2Details) (*model.OAuth2Token, error) {
	existToken, err := tokenService.tokenStore.GetAccessToken(oauth2Details)
	var refreshToken *model.OAuth2Token
	if err == nil {
		if !existToken.IsExpired() {
			tokenService.tokenStore.StoreAccessToken(existToken, oauth2Details)
			return existToken, nil
		} else {
			tokenService.tokenStore.RemoveAccessToken(existToken.TokenValue)
			if existToken.RefreshToken != nil {
				refreshToken = existToken.RefreshToken
				tokenService.tokenStore.RemoveRefreshToken(refreshToken.TokenValue)
			}
		}
	}
	// 刷新时间和创建时间不一样
	if refreshToken == nil || refreshToken.IsExpired() {
		refreshToken, err = tokenService.createRefreshToken(oauth2Details)
		if err != nil {
			return nil, err
		}
	}
	// 生成新的访问令牌
	accessToken, err := tokenService.createAccessToken(refreshToken, oauth2Details)
	if err == nil {
		// 保存新生成令牌
		tokenService.tokenStore.StoreAccessToken(accessToken, oauth2Details)
		tokenService.tokenStore.StoreRefreshToken(refreshToken, oauth2Details)
	}
	return accessToken, err
}

func (tokenService *DefaultTokenService) createAccessToken(refreshToken *model.OAuth2Token, oauth2Detail *model.OAuth2Details) (*model.OAuth2Token, error) {
	validitySeconds := oauth2Detail.Client.AccessTokenValiditySeconds
	s, _ := time.ParseDuration(strconv.Itoa(validitySeconds) + "s")
	expiredTime := time.Now().Add(s)
	access := &model.OAuth2Token{
		RefreshToken: refreshToken,
		ExpriesTime:  &expiredTime,
		TokenValue:   uuid.NewV4().String(),
	}
	if tokenService.tokenEnhancer != nil {
		return tokenService.tokenEnhancer.Enhance(access, oauth2Detail)
	}
	return access, nil
}

func (tokenService *DefaultTokenService) createRefreshToken(oauth2Details *model.OAuth2Details) (*model.OAuth2Token, error) {
	validitySeconds := oauth2Details.Client.RefreshTokenValiditySeconds
	s, _ := time.ParseDuration(strconv.Itoa(validitySeconds) + "s")
	expiredTime := time.Now().Add(s)
	refreshToken := &model.OAuth2Token{
		ExpriesTime: &expiredTime,
		TokenValue:  uuid.NewV4().String(),
	}

	if tokenService.tokenEnhancer != nil {
		return tokenService.tokenEnhancer.Enhance(refreshToken, oauth2Details)
	}
	return refreshToken, nil
}

func (tokenService *DefaultTokenService) RefreshAccessToken(refreshTokenValue string) (*model.OAuth2Token, error) {
	refreshToken, err := tokenService.tokenStore.ReadRefreshToken(refreshTokenValue)
	if err == nil {
		if refreshToken.IsExpired() {
			return nil, ErrExpiredToken
		}
		oauth2Details, err := tokenService.tokenStore.ReadOAuth2DetailsForRefreshToken(refreshTokenValue)
		if err == nil {
			oauth2Token, err := tokenService.tokenStore.GetAccessToken(oauth2Details)
			if err == nil {
				tokenService.tokenStore.RemoveAccessToken(oauth2Token.TokenValue)
			}
			// 移除已使用的刷新令牌
			tokenService.tokenStore.RemoveRefreshToken(refreshTokenValue)
			refreshToken, err = tokenService.createRefreshToken(oauth2Details)
			if err == nil {
				accessToken, err := tokenService.createAccessToken(refreshToken, oauth2Details)
				if err == nil {
					tokenService.tokenStore.StoreAccessToken(accessToken, oauth2Details)
					tokenService.tokenStore.StoreRefreshToken(refreshToken, oauth2Details)
				}
				return accessToken, err
			}
		}
	}
	return nil, err
}

func (tokenService *DefaultTokenService) GetAccessToken(details *model.OAuth2Details) (*model.OAuth2Token, error) {
	return tokenService.tokenStore.GetAccessToken(details)
}

func (tokenService *DefaultTokenService) ReadAccessToken(tokenValue string) (*model.OAuth2Token, error) {
	return tokenService.tokenStore.ReadAccessToken(tokenValue)
}

type TokenEnhancer interface {
	// 组装 Token 信息
	Enhance(oauth2Token *model.OAuth2Token, oauth2Details *model.OAuth2Details) (*model.OAuth2Token, error)
	// 从 Token 中还原信息
	Extract(tokenValue string) (*model.OAuth2Token, *model.OAuth2Details, error)
}

type TokenStore interface {

	// 存储访问令牌
	StoreAccessToken(oauth2Token *model.OAuth2Token, oauth2Details *model.OAuth2Details)
	// 根据令牌值获取访问令牌结构体
	ReadAccessToken(tokenValue string) (*model.OAuth2Token, error)
	// 根据令牌值获取令牌对应的客户端和用户信息
	ReadOAuth2Details(tokenValue string) (*model.OAuth2Details, error)
	// 根据客户端信息和用户信息获取访问令牌
	GetAccessToken(oauth2Details *model.OAuth2Details) (*model.OAuth2Token, error)
	// 移除存储的访问令牌
	RemoveAccessToken(tokenValue string)
	// 存储刷新令牌
	StoreRefreshToken(oauth2Token *model.OAuth2Token, oauth2Details *model.OAuth2Details)
	// 移除存储的刷新令牌
	RemoveRefreshToken(oauth2Token string)
	// 根据令牌值获取刷新令牌
	ReadRefreshToken(tokenValue string) (*model.OAuth2Token, error)
	// 根据令牌值获取刷新令牌对应的客户端和用户信息
	ReadOAuth2DetailsForRefreshToken(tokenValue string) (*model.OAuth2Details, error)
}

type JwtTokenEnhancer struct {
	secretKey []byte
}

func NewJwtTokenEnhancer(secretKey string) TokenEnhancer {
	return &JwtTokenEnhancer{
		secretKey: []byte(secretKey),
	}
}

func (enhancer *JwtTokenEnhancer) Extract(tokenValue string) (*model.OAuth2Token, *model.OAuth2Details, error) {
	token, err := jwt.ParseWithClaims(tokenValue, &OAuth2TokenCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return enhancer.secretKey, nil
	})
	if err == nil {
		claims := token.Claims.(*OAuth2TokenCustomClaims)
		expiresTime := time.Unix(claims.ExpiresAt, 0)
		return &model.OAuth2Token{
				RefreshToken: &claims.RefreshToken,
				TokenType:    tokenValue,
				ExpriesTime:  &expiresTime,
			}, &model.OAuth2Details{
				User:   &claims.UserDetails,
				Client: &claims.ClientDetails,
			}, nil
	}
	return nil, nil, err
}

func (enhancer *JwtTokenEnhancer) Enhance(oauth2Token *model.OAuth2Token, oauth2Details *model.OAuth2Details) (*model.OAuth2Token, error) {
	return enhancer.sign(oauth2Token, oauth2Details)
}

type OAuth2TokenCustomClaims struct {
	UserDetails   model.UserDetails
	ClientDetails model.ClientDetails
	RefreshToken  model.OAuth2Token
	//内嵌模式
	jwt.StandardClaims
}

func (enhancer *JwtTokenEnhancer) sign(oauth2Token *model.OAuth2Token, oauth2Details *model.OAuth2Details) (*model.OAuth2Token, error) {
	expireTime := oauth2Token.ExpriesTime
	clientDetails := *oauth2Details.Client
	userDetails := *oauth2Details.User
	clientDetails.ClientSecret = ""
	userDetails.Password = ""
	claims := &OAuth2TokenCustomClaims{
		UserDetails:   userDetails,
		ClientDetails: clientDetails,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "System",
		},
	}
	if oauth2Token.RefreshToken != nil {
		claims.RefreshToken = *oauth2Token.RefreshToken
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenValue, err := token.SignedString(enhancer.secretKey)
	if err == nil {
		oauth2Token.TokenValue = tokenValue
		oauth2Token.TokenType = "jwt"
		return oauth2Token, nil

	}
	return nil, err
}

type JwtTokenStore struct {
	jwtTokenEnhancer *JwtTokenEnhancer
}

func NewJwtTokenStore(jwtTokenEnhancer *JwtTokenEnhancer) *JwtTokenStore {
	return &JwtTokenStore{
		jwtTokenEnhancer: jwtTokenEnhancer,
	}
}

func (JwtTokenStore) StoreAccessToken(oauth2Token *model.OAuth2Token, oauth2Details *model.OAuth2Details) {
	panic("implement me")
}

func (jwtTokenStore *JwtTokenStore) ReadAccessToken(tokenValue string) (*model.OAuth2Token, error) {
	oauth2Token, _, err := jwtTokenStore.jwtTokenEnhancer.Extract(tokenValue)
	if err == nil {
		return oauth2Token, nil
	}
	return nil, err
}

func (jwtTokenStore *JwtTokenStore) ReadOAuth2Details(tokenValue string) (*model.OAuth2Details, error) {
	_, oauth2Details, err := jwtTokenStore.jwtTokenEnhancer.Extract(tokenValue)
	return oauth2Details, err
}

func (jwtTokenStore *JwtTokenStore) GetAccessToken(oauth2Details *model.OAuth2Details) (*model.OAuth2Token, error) {
	return nil, ErrNotSupportOperation
}
func (jwtTokenStore *JwtTokenStore) RemoveAccessToken(tokenValue string) {

}

func (jwtTokenStore *JwtTokenStore) StoreRefreshToken(oauth2Token *model.OAuth2Token, oauth2Details *model.OAuth2Details) {
	panic("implement me")
}

func (jwtTokenStore *JwtTokenStore) RemoveRefreshToken(oauth2Token string) {
	panic("implement me")
}

func (jwtTokenStore *JwtTokenStore) ReadRefreshToken(tokenValue string) (*model.OAuth2Token, error) {
	oauth2Token, _, err := jwtTokenStore.jwtTokenEnhancer.Extract(tokenValue)
	return oauth2Token, err
}

func (jwtTokenStore *JwtTokenStore) ReadOAuth2DetailsForRefreshToken(tokenValue string) (*model.OAuth2Details, error) {
	_, oauth2Details, err := jwtTokenStore.jwtTokenEnhancer.Extract(tokenValue)
	return oauth2Details, err
}
