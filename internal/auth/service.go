package auth

import "fmt"

type Service interface {
	Name() string
	Login() error
	Logout() error
	IsLoggedIn() bool
}

var registry = map[string]Service{}

func init() {
	Register(NewTrelloService())
}

func Register(svc Service) {
	registry[svc.Name()] = svc
}

func Get(name string) (Service, error) {
	svc, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown service: %s\nAvailable services: %s", name, AvailableNames())
	}
	return svc, nil
}

func AvailableNames() string {
	names := ""
	for name := range registry {
		if names != "" {
			names += ", "
		}
		names += name
	}
	return names
}
