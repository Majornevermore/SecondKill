package model

type ClientDetails struct {
	// Client 标识
	ClientId     string
	ClientSecret string
	// 访问令牌有效时间，秒
	AccessTokenValiditySeconds int
	// 刷新令牌有效时间，秒
	RefreshTokenValiditySeconds int
	// 重定向地址，授权码类型中使用
	RegisteredRedirectUri string
	// 可以使用的授权类型
	AuthorizedGrantTypes []string
}

func (clientDetails *ClientDetails) IsMatch(clientId string, clientSecret string) bool {
	return clientId == clientDetails.ClientId && clientSecret == clientDetails.ClientSecret
}

type ClientDetailsModel struct {
}
