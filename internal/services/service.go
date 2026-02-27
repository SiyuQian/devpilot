package services

type Service interface {
	Name() string
	Login() error
	Logout() error
	IsLoggedIn() bool
}
