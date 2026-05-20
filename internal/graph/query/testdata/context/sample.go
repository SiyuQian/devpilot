package sample

func Greet(name string) string {
	return "hi " + name
}

func CallGreet() string {
	return Greet("world")
}
