package main

import (
	"fmt"
	"sync"
)

var (
	ffiLibs   = make(map[int]*ffiHandle)
	ffiLibIdx int
	ffiMu     sync.Mutex
)

type ffiHandle struct {
	name string
	lib  uintptr
	funcs map[string]uintptr
}

func (e *Eval) builtinCgoOpen(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("cgo/open requires 1 argument (path)")
	}
	path, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("cgo/open: path must be a string")
	}

	handle, err := dlopen(string(path))
	if err != nil {
		return nil, fmt.Errorf("cgo/open: %v", err)
	}

	ffiMu.Lock()
	ffiLibIdx++
	id := ffiLibIdx
	ffiLibs[id] = &ffiHandle{
		name:   string(path),
		lib:    handle,
		funcs:  make(map[string]uintptr),
	}
	ffiMu.Unlock()

	return &CgoLib{Name: fmt.Sprintf("%d", id)}, nil
}

func (e *Eval) builtinCgoFunc(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("cgo/func requires 2 arguments (lib fn-name)")
	}
	lib, ok := args[0].(*CgoLib)
	if !ok {
		return nil, fmt.Errorf("cgo/func: first argument must be a cgo-lib")
	}
	fnName, ok := args[1].(String)
	if !ok {
		return nil, fmt.Errorf("cgo/func: function name must be a string")
	}

	id := atoi(lib.Name)
	ffiMu.Lock()
	handle, ok := ffiLibs[id]
	ffiMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("cgo/func: unknown library")
	}

	fnPtr, err := dlsym(handle.lib, string(fnName))
	if err != nil {
		return nil, fmt.Errorf("cgo/func: %v", err)
	}

	ffiMu.Lock()
	handle.funcs[string(fnName)] = fnPtr
	ffiMu.Unlock()

	return Nil, nil
}

func (e *Eval) builtinCgoCall(args []Value) (Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("cgo/call requires at least 2 arguments (lib fn-name [args...])")
	}
	lib, ok := args[0].(*CgoLib)
	if !ok {
		return nil, fmt.Errorf("cgo/call: first argument must be a cgo-lib")
	}
	fnName, ok := args[1].(String)
	if !ok {
		return nil, fmt.Errorf("cgo/call: function name must be a string")
	}

	id := atoi(lib.Name)
	ffiMu.Lock()
	handle, ok := ffiLibs[id]
	ffiMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("cgo/call: unknown library")
	}

	fnPtr, ok := handle.funcs[string(fnName)]
	if !ok {
		return nil, fmt.Errorf("cgo/call: function '%s' not resolved (use cgo/func first)", string(fnName))
	}

	callArgs := make([]uintptr, 0, len(args)-2)
	for _, a := range args[2:] {
		callArgs = append(callArgs, valueToUintptr(a))
	}

	result, err := callFunc(fnPtr, callArgs)
	if err != nil {
		return nil, fmt.Errorf("cgo/call: %v", err)
	}

	return uintptrToValue(result), nil
}

func (e *Eval) builtinCgoClose(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("cgo/close requires 1 argument")
	}
	lib, ok := args[0].(*CgoLib)
	if !ok {
		return nil, fmt.Errorf("cgo/close: argument must be a cgo-lib")
	}

	id := atoi(lib.Name)
	ffiMu.Lock()
	handle, ok := ffiLibs[id]
	if ok {
		delete(ffiLibs, id)
	}
	ffiMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("cgo/close: unknown library")
	}

	if err := dlclose(handle.lib); err != nil {
		return nil, fmt.Errorf("cgo/close: %v", err)
	}
	return Nil, nil
}

func valueToUintptr(v Value) uintptr {
	switch val := v.(type) {
	case Integer:
		return uintptr(int64(val))
	case Float:
		return uintptr(int64(val))
	case Boolean:
		if val {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func uintptrToValue(v uintptr) Value {
	return Integer(int64(v))
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
