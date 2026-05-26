//go:build !cgo

package main

import "errors"

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
