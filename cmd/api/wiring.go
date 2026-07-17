package main

import (
	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/ports/in"
	"github.com/quii/ce/internal/ports/out"
)

// docs/adr/0025-composition-root.md: the only place a concrete out-adapter gets constructed.
type OutPorts interface {
	out.GreetingFinder
}

type outPorts struct {
	out.GreetingFinder
}

func NewOutPorts() OutPorts {
	return &outPorts{
		GreetingFinder: memory.NewGreetingFinder(),
	}
}

type Application struct {
	GetGreeting in.Greeter
}

func NewApplication(ports OutPorts) *Application {
	return &Application{
		GetGreeting: in.NewGetGreetingUseCase(ports),
	}
}
