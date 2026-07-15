package domain

import "strings"

type Greeting string

type Name string

func NewName(raw string) Name {
	return Name(strings.TrimSpace(raw))
}

func (n Name) IsBlank() bool {
	return n == ""
}

func (n Name) Greet() Greeting {
	return Greeting("Hello, " + string(n) + "!")
}
