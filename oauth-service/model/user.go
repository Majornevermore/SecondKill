package model

type UserDetails struct {
	UserId   int64
	Username string
	Password string
	// 具备权限
	Authorities []string
}

func (userDetail *UserDetails) IsMatch(username string, password string) bool {
	return userDetail.Username == username && userDetail.Password == password
}
