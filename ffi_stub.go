//go:build !cgo

package main

import (
	"errors"
	"unsafe"
)

func dlopen(path string) (uintptr, error) {
	return 0, errors.New("cgo/open requires CGO_ENABLED=1 (install gcc and rebuild)")
}

func dlsym(handle uintptr, name string) (uintptr, error) {
	return 0, errors.New("cgo/func requires CGO_ENABLED=1")
}

func dlclose(handle uintptr) error {
	return errors.New("cgo/close requires CGO_ENABLED=1")
}

func callFunc(fnPtr uintptr, args []uintptr) (uintptr, error) {
	return 0, errors.New("cgo/call requires CGO_ENABLED=1")
}

func callFuncDouble(fnPtr uintptr, args []float64) (float64, error) {
	return 0, errors.New("cgo/call requires CGO_ENABLED=1")
}

func allocCString(s string) unsafe.Pointer {
	return nil
}

func freeCString(p unsafe.Pointer) {}

func goStringFromPtr(p uintptr) string {
	return ""
}

func callFuncRetString(fnPtr uintptr, args []uintptr) (string, error) {
	return "", errors.New("cgo/call requires CGO_ENABLED=1")
}

func callFuncVoid(fnPtr uintptr, args []uintptr) error {
	return errors.New("cgo/call requires CGO_ENABLED=1")
}

func callFuncWithStrings(fnPtr uintptr, strArgs []unsafe.Pointer, intArgs []uintptr) (uintptr, error) {
	return 0, errors.New("cgo/call requires CGO_ENABLED=1")
}

func callFuncDoubleWithStringArg(fnPtr uintptr, strArg unsafe.Pointer) (float64, error) {
	return 0, errors.New("cgo/call requires CGO_ENABLED=1")
}

func callFuncDoubleWithIntArg(fnPtr uintptr, intArg uintptr) (float64, error) {
	return 0, errors.New("cgo/call requires CGO_ENABLED=1")
}
