package main

import (
	"fmt"
	alias "strings"
)

func Greet(name string) string {
	return fmt.Sprintf("hi %s", name)
}

func main() {
	fmt.Println(Greet("world"))
}

type Greeter struct{ prefix string }

func (g *Greeter) Hello(name string) string { return g.prefix + " " + name }
func (g Greeter) silent() string             { return "" }

type Greeter2 struct{}
type Hello interface {
	Greet() string
}
type Alias = string
type IntPtr *int

var _ = alias.ToUpper
