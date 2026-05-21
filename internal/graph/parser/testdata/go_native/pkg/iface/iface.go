package iface

type Speaker interface {
	Speak() string
}

// Alias exercises the type-alias path for the native parser.
type Alias = string
