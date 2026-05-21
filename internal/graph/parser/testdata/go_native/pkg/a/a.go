package a

func Greet(name string) string {
	return "hi " + name
}

func Run() string {
	return Greet("world")
}
