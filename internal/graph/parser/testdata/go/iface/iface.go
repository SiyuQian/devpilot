package iface

type Greeter interface {
	Greet() string
}

type Console struct{}

func (Console) Greet() string { return "hello" }

type Mute struct{}
