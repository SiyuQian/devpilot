package auth

import "fmt"

// Service is the interface implemented by each external service that
// participates in devpilot's login, logout, and status commands.
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

// Register adds svc to the auth registry, keyed by its Name. A later call
// with the same name replaces the earlier entry.
func Register(svc Service) {
	registry[svc.Name()] = svc
}

// Get returns the registered service with the given name, or an error if no
// such service is registered.
func Get(name string) (Service, error) {
	svc, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown service: %s\nAvailable services: %s", name, AvailableNames())
	}
	return svc, nil
}

// AvailableNames returns a human-readable, comma-separated list of all
// registered service names.
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
