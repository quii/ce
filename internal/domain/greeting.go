package domain

import "strings"

type Greeting string

type Prefix string

type Name string

func NewName(raw string) Name {
	return Name(strings.TrimSpace(raw))
}

func (n Name) IsBlank() bool {
	return n == ""
}

func (n Name) Greet(prefix Prefix) Greeting {
	if n.IsBlank() {
		n = "World"
	}
	return Greeting(string(prefix) + ", " + string(n) + "!")
}
