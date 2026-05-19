package main

import "fmt"

func Greet(name string) string {
	return fmt.Sprintf("hi %s", name)
}

func main() {
	fmt.Println(Greet("world"))
}
