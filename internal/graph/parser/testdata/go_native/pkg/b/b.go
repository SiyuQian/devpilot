package b

import "example.com/native/pkg/a"

func B() string {
	return a.Greet("y")
}
