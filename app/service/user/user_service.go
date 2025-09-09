package user

// interface
type UserService interface {
	//创建
	CreateUser() (bool, error)
}
