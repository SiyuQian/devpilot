package a

func Greet(name string) string {
	return "hi " + name
}

func Run() string {
	return Greet("world")
}

// UsesLen exists to verify that calls to builtins (len) do not emit edges.
func UsesLen() int {
	return len("hi")
}
