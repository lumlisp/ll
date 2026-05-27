package main

import (
	"fmt"
	"math"
	"sync"
	"unsafe"
)

// ffiType represents the type of an FFI argument or return value
type ffiType int

const (
	ffiAuto    ffiType = iota // auto-detect based on value type
	ffiInt                    // int64 -> C long
	ffiDouble                 // float64 -> C double
	ffiString                 // string -> C char*
	ffiPointer                // vector/pointer -> C void*
	ffiVoid                   // no return value
)

func parseFFIType(s string) ffiType {
	switch s {
	case "int":
		return ffiInt
	case "double":
		return ffiDouble
	case "string":
		return ffiString
	case "pointer", "ptr":
		return ffiPointer
	case "void":
		return ffiVoid
	default:
		return ffiAuto
	}
}

// ffiFuncInfo stores a resolved function pointer with optional type info
type ffiFuncInfo struct {
	ptr      uintptr
	argTypes []ffiType
	retType  ffiType
}

var (
	ffiLibs   = make(map[int]*ffiHandle)
	ffiLibIdx int
	ffiMu     sync.Mutex
)

type ffiHandle struct {
	name  string
	lib   uintptr
	funcs map[string]*ffiFuncInfo
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
		funcs:  make(map[string]*ffiFuncInfo),
	}
	ffiMu.Unlock()

	return &CgoLib{Name: fmt.Sprintf("%d", id)}, nil
}

func (e *Eval) builtinCgoFunc(args []Value) (Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("cgo/func requires 2-3 arguments (lib fn-name [type-signature])")
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

	info := &ffiFuncInfo{ptr: fnPtr}

	// Parse optional type signature: '(arg-types... ret-type)
	// Last element is the return type, rest are arg types
	if len(args) >= 3 {
		sigList, ok := args[2].(*Cons)
		if !ok {
			return nil, fmt.Errorf("cgo/func: type signature must be a list, e.g. '(string int)")
		}
		sigSlice, ok := ListToSlice(sigList)
		if !ok {
			return nil, fmt.Errorf("cgo/func: invalid type signature")
		}
		if len(sigSlice) < 1 {
			return nil, fmt.Errorf("cgo/func: type signature must have at least return type")
		}

		// Last element is return type
		retSym, ok := sigSlice[len(sigSlice)-1].(*Sym)
		if !ok {
			return nil, fmt.Errorf("cgo/func: return type must be a symbol")
		}
		info.retType = parseFFIType(retSym.Name)

		// All elements before last are arg types
		info.argTypes = make([]ffiType, len(sigSlice)-1)
		for i, at := range sigSlice[:len(sigSlice)-1] {
			sym, ok := at.(*Sym)
			if !ok {
				return nil, fmt.Errorf("cgo/func: arg type must be a symbol")
			}
			info.argTypes[i] = parseFFIType(sym.Name)
		}
	}

	ffiMu.Lock()
	handle.funcs[string(fnName)] = info
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

	fnInfo, ok := handle.funcs[string(fnName)]
	if !ok {
		return nil, fmt.Errorf("cgo/call: function '%s' not resolved (use cgo/func first)", string(fnName))
	}

	callArgs := args[2:]

	// If type info is specified, use typed dispatch
	if fnInfo.argTypes != nil {
		return callWithTypes(fnInfo, callArgs)
	}

	// Legacy auto-detect mode (backward compatible)
	hasFloat := false
	for _, a := range callArgs {
		if _, ok := a.(Float); ok {
			hasFloat = true
			break
		}
	}

	if hasFloat {
		dblArgs := make([]float64, 0, len(callArgs))
		for _, a := range callArgs {
			switch v := a.(type) {
			case Float:
				dblArgs = append(dblArgs, float64(v))
			case Integer:
				dblArgs = append(dblArgs, float64(v))
			default:
				dblArgs = append(dblArgs, 0)
			}
		}
		result, err := callFuncDouble(fnInfo.ptr, dblArgs)
		if err != nil {
			return nil, fmt.Errorf("cgo/call: %v", err)
		}
		return Float(result), nil
	}

	uintArgs := make([]uintptr, 0, len(callArgs))
	for _, a := range callArgs {
		uintArgs = append(uintArgs, valueToUintptr(a))
	}

	result, err := callFunc(fnInfo.ptr, uintArgs)
	if err != nil {
		return nil, fmt.Errorf("cgo/call: %v", err)
	}

	return uintptrToValue(result), nil
}

// callWithTypes handles FFI calls using explicit type information
func callWithTypes(info *ffiFuncInfo, args []Value) (Value, error) {
	// Prepare argument arrays
	stringArgs := make([]unsafe.Pointer, 0)
	intArgs := make([]uintptr, 0)
	stringArgIdx := make([]int, 0) // indices of string args in the original list

	for i, arg := range args {
		t := ffiInt
		if i < len(info.argTypes) {
			t = info.argTypes[i]
		}

		switch t {
		case ffiString:
			s, ok := arg.(String)
			if !ok {
				return nil, fmt.Errorf("cgo/call: expected string for arg %d", i)
			}
			cstr := allocCString(string(s))
			stringArgs = append(stringArgs, cstr)
			intArgs = append(intArgs, 0) // placeholder
			stringArgIdx = append(stringArgIdx, i)
		case ffiPointer:
			switch v := arg.(type) {
			case String:
				cstr := allocCString(string(v))
				stringArgs = append(stringArgs, cstr)
				intArgs = append(intArgs, 0)
				stringArgIdx = append(stringArgIdx, i)
			case *Vector:
				// For vectors, pass a pointer (for now, pass 0 — full array support TBD)
				intArgs = append(intArgs, 0)
			default:
				intArgs = append(intArgs, valueToUintptr(arg))
			}
		case ffiDouble:
			switch v := arg.(type) {
			case Float:
				intArgs = append(intArgs, uintptr(math.Float64bits(float64(v))))
			case Integer:
				intArgs = append(intArgs, uintptr(math.Float64bits(float64(v))))
			default:
				intArgs = append(intArgs, 0)
			}
		default: // ffiInt, ffiAuto
			intArgs = append(intArgs, valueToUintptr(arg))
		}
	}

	// Schedule cleanup of C strings
	defer func() {
		for _, p := range stringArgs {
			freeCString(p)
		}
	}()

	// Build flat uintptr arg list preserving argument order
	flatArgs := make([]uintptr, len(args))
	si := 0
	ii := 0
	for i := 0; i < len(args); i++ {
		if si < len(stringArgIdx) && stringArgIdx[si] == i {
			flatArgs[i] = uintptr(stringArgs[si])
			si++
		} else {
			flatArgs[i] = intArgs[ii]
			ii++
		}
	}

	// Dispatch based on return type
	switch info.retType {
	case ffiString:
		result, err := callFuncRetString(info.ptr, flatArgs)
		if err != nil {
			return nil, err
		}
		return String(result), nil

	case ffiVoid:
		err := callFuncVoid(info.ptr, flatArgs)
		if err != nil {
			return nil, err
		}
		return Nil, nil

	case ffiDouble:
		// Handle mixed argument types for double return
		if len(args) == 1 && len(info.argTypes) == 1 && info.argTypes[0] == ffiString {
			result, err := callFuncDoubleWithStringArg(info.ptr, stringArgs[0])
			if err != nil {
				return nil, err
			}
			return Float(result), nil
		}
		if len(args) == 1 && len(info.argTypes) == 1 && info.argTypes[0] == ffiInt {
			result, err := callFuncDoubleWithIntArg(info.ptr, flatArgs[0])
			if err != nil {
				return nil, err
			}
			return Float(result), nil
		}
		// Fallback: convert all args to double (works for pure double args)
		dblArgs := make([]float64, len(flatArgs))
		for i, ua := range flatArgs {
			dblArgs[i] = math.Float64frombits(uint64(ua))
		}
		result, err := callFuncDouble(info.ptr, dblArgs)
		if err != nil {
			return nil, err
		}
		return Float(result), nil

	case ffiPointer:
		result, err := callFunc(info.ptr, flatArgs)
		if err != nil {
			return nil, err
		}
		return Integer(int64(result)), nil

	default: // ffiInt, ffiAuto
		result, err := callFunc(info.ptr, flatArgs)
		if err != nil {
			return nil, err
		}
		return uintptrToValue(result), nil
	}
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
		return uintptr(math.Float64bits(float64(val)))
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
