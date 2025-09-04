package user

type UserDao interface {
	//创建
	Create() (bool, error)
}
