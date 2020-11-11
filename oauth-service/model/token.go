package model

import "time"

type OAuth2Token struct {
	RefreshToken *OAuth2Token
	//令牌类型
	TokenType   string
	TokenValue  string
	ExpriesTime *time.Time
}

func (oauth2token *OAuth2Token) IsExpired() bool {
	return oauth2token.ExpriesTime != nil &&
		oauth2token.ExpriesTime.Before(time.Now())
}

type OAuth2Details struct {
	Client *ClientDetails
	User   *UserDetails
}
