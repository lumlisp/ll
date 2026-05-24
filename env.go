package main

import "fmt"

type Env struct {
	parent *Env
	values map[string]Value
}

func NewEnv(parent *Env) *Env {
	return &Env{
		parent: parent,
		values: make(map[string]Value),
	}
}

func (e *Env) Set(name string, val Value) {
	e.values[name] = val
}

func (e *Env) Get(name string) (Value, error) {
	if v, ok := e.values[name]; ok {
		return v, nil
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return nil, fmt.Errorf("undefined variable: %s", name)
}

func (e *Env) Has(name string) bool {
	if _, ok := e.values[name]; ok {
		return true
	}
	if e.parent != nil {
		return e.parent.Has(name)
	}
	return false
}

func (e *Env) SetMutate(name string, val Value) error {
	if _, ok := e.values[name]; ok {
		e.values[name] = val
		return nil
	}
	if e.parent != nil {
		return e.parent.SetMutate(name, val)
	}
	return fmt.Errorf("cannot set! undefined variable: %s", name)
}

func (e *Env) Extend() *Env {
	return NewEnv(e)
}
