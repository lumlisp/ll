//go:build cgo

package main

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

// Integer calling convention (long args, long return)
typedef long (*fn0)(void);
typedef long (*fn1)(long);
typedef long (*fn2)(long, long);
typedef long (*fn3)(long, long, long);
typedef long (*fn4)(long, long, long, long);
typedef long (*fn5)(long, long, long, long, long);
typedef long (*fn6)(long, long, long, long, long, long);

long call_fn0(void *f) { return ((fn0)f)(); }
long call_fn1(void *f, long a) { return ((fn1)f)(a); }
long call_fn2(void *f, long a, long b) { return ((fn2)f)(a, b); }
long call_fn3(void *f, long a, long b, long c) { return ((fn3)f)(a, b, c); }
long call_fn4(void *f, long a, long b, long c, long d) { return ((fn4)f)(a, b, c, d); }
long call_fn5(void *f, long a, long b, long c, long d, long e) { return ((fn5)f)(a, b, c, d, e); }
long call_fn6(void *f, long a, long b, long c, long d, long e, long f2) { return ((fn6)f)(a, b, c, d, e, f2); }

// Float calling convention (double args, double return)
typedef double (*fnd0)(void);
typedef double (*fnd1)(double);
typedef double (*fnd2)(double, double);
typedef double (*fnd3)(double, double, double);
typedef double (*fnd4)(double, double, double, double);
typedef double (*fnd5)(double, double, double, double, double);
typedef double (*fnd6)(double, double, double, double, double, double);

double call_fnd0(void *f) { return ((fnd0)f)(); }
double call_fnd1(void *f, double a) { return ((fnd1)f)(a); }
double call_fnd2(void *f, double a, double b) { return ((fnd2)f)(a, b); }
double call_fnd3(void *f, double a, double b, double c) { return ((fnd3)f)(a, b, c); }
double call_fnd4(void *f, double a, double b, double c, double d) { return ((fnd4)f)(a, b, c, d); }
double call_fnd5(void *f, double a, double b, double c, double d, double e) { return ((fnd5)f)(a, b, c, d, e); }
double call_fnd6(void *f, double a, double b, double c, double d, double e, double f2) { return ((fnd6)f)(a, b, c, d, e, f2); }

// String return calling convention (long args, char* return)
typedef char* (*fnr0)(void);
typedef char* (*fnr1)(long);
typedef char* (*fnr2)(long, long);
typedef char* (*fnr3)(long, long, long);
typedef char* (*fnr4)(long, long, long, long);
typedef char* (*fnr5)(long, long, long, long, long);
typedef char* (*fnr6)(long, long, long, long, long, long);

char* call_fn_ret0(void *f) { return ((fnr0)f)(); }
char* call_fn_ret1(void *f, long a) { return ((fnr1)f)(a); }
char* call_fn_ret2(void *f, long a, long b) { return ((fnr2)f)(a, b); }
char* call_fn_ret3(void *f, long a, long b, long c) { return ((fnr3)f)(a, b, c); }
char* call_fn_ret4(void *f, long a, long b, long c, long d) { return ((fnr4)f)(a, b, c, d); }
char* call_fn_ret5(void *f, long a, long b, long c, long d, long e) { return ((fnr5)f)(a, b, c, d, e); }
char* call_fn_ret6(void *f, long a, long b, long c, long d, long e, long f2) { return ((fnr6)f)(a, b, c, d, e, f2); }

// String arg + int return calling convention (char* args, long return)
typedef long (*fns1)(char*);
typedef long (*fns2)(char*, long);
typedef long (*fns3)(char*, long, long);
typedef long (*fns4)(char*, long, long, long);

long call_fn_s1(void *f, char* a) { return ((fns1)f)(a); }
long call_fn_s2(void *f, char* a, long b) { return ((fns2)f)(a, b); }
long call_fn_s3(void *f, char* a, long b, long c) { return ((fns3)f)(a, b, c); }
long call_fn_s4(void *f, char* a, long b, long c, long d) { return ((fns4)f)(a, b, c, d); }

// String arg + double return calling convention (char* arg, double return)
typedef double (*fnd_s1)(char*);
typedef double (*fnd_i1)(long);

double call_fnd_s1(void *f, char* a) { return ((fnd_s1)f)(a); }
double call_fnd_i1(void *f, long a) { return ((fnd_i1)f)(a); }
*/
import "C"
import (
	"errors"
	"unsafe"
)

func dlopen(path string) (uintptr, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	handle := C.dlopen(cpath, C.RTLD_LAZY|C.RTLD_LOCAL)
	if handle == nil {
		errStr := C.GoString(C.dlerror())
		return 0, errors.New(errStr)
	}
	return uintptr(unsafe.Pointer(handle)), nil
}

func dlsym(handle uintptr, name string) (uintptr, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	ptr := C.dlsym(unsafe.Pointer(handle), cname)
	if ptr == nil {
		errStr := C.GoString(C.dlerror())
		return 0, errors.New(errStr)
	}
	return uintptr(unsafe.Pointer(ptr)), nil
}

func callFuncDouble(fnPtr uintptr, args []float64) (float64, error) {
	fn := unsafe.Pointer(fnPtr)
	switch len(args) {
	case 0:
		return float64(C.call_fnd0(fn)), nil
	case 1:
		return float64(C.call_fnd1(fn, C.double(args[0]))), nil
	case 2:
		return float64(C.call_fnd2(fn, C.double(args[0]), C.double(args[1]))), nil
	case 3:
		return float64(C.call_fnd3(fn, C.double(args[0]), C.double(args[1]), C.double(args[2]))), nil
	case 4:
		return float64(C.call_fnd4(fn, C.double(args[0]), C.double(args[1]), C.double(args[2]), C.double(args[3]))), nil
	case 5:
		return float64(C.call_fnd5(fn, C.double(args[0]), C.double(args[1]), C.double(args[2]), C.double(args[3]), C.double(args[4]))), nil
	case 6:
		return float64(C.call_fnd6(fn, C.double(args[0]), C.double(args[1]), C.double(args[2]), C.double(args[3]), C.double(args[4]), C.double(args[5]))), nil
	default:
		return 0, errors.New("cgo/call: too many arguments for float function (max 6)")
	}
}

func dlclose(handle uintptr) error {
	ret := C.dlclose(unsafe.Pointer(handle))
	if ret != 0 {
		errStr := C.GoString(C.dlerror())
		return errors.New(errStr)
	}
	return nil
}

// String allocation helpers for FFI typed calls
func allocCString(s string) unsafe.Pointer {
	return unsafe.Pointer(C.CString(s))
}

func freeCString(p unsafe.Pointer) {
	C.free(p)
}

func goStringFromPtr(p uintptr) string {
	return C.GoString((*C.char)(unsafe.Pointer(p)))
}

// callFuncRetString calls a C function that returns char* (string)
func callFuncRetString(fnPtr uintptr, args []uintptr) (string, error) {
	fn := unsafe.Pointer(fnPtr)
	switch len(args) {
	case 0:
		return C.GoString(C.call_fn_ret0(fn)), nil
	case 1:
		return C.GoString(C.call_fn_ret1(fn, C.long(args[0]))), nil
	case 2:
		return C.GoString(C.call_fn_ret2(fn, C.long(args[0]), C.long(args[1]))), nil
	case 3:
		return C.GoString(C.call_fn_ret3(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]))), nil
	case 4:
		return C.GoString(C.call_fn_ret4(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]), C.long(args[3]))), nil
	case 5:
		return C.GoString(C.call_fn_ret5(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]), C.long(args[3]), C.long(args[4]))), nil
	case 6:
		return C.GoString(C.call_fn_ret6(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]), C.long(args[3]), C.long(args[4]), C.long(args[5]))), nil
	default:
		return "", errors.New("cgo/call: too many arguments (max 6)")
	}
}

// callFuncVoid calls a C function that returns void (calls through int convention, discards result)
func callFuncVoid(fnPtr uintptr, args []uintptr) error {
	fn := unsafe.Pointer(fnPtr)
	switch len(args) {
	case 0:
		C.call_fn0(fn)
		return nil
	case 1:
		C.call_fn1(fn, C.long(args[0]))
		return nil
	case 2:
		C.call_fn2(fn, C.long(args[0]), C.long(args[1]))
		return nil
	case 3:
		C.call_fn3(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]))
		return nil
	case 4:
		C.call_fn4(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]), C.long(args[3]))
		return nil
	case 5:
		C.call_fn5(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]), C.long(args[3]), C.long(args[4]))
		return nil
	case 6:
		C.call_fn6(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]), C.long(args[3]), C.long(args[4]), C.long(args[5]))
		return nil
	default:
		return errors.New("cgo/call: too many arguments (max 6)")
	}
}

// callFuncDoubleWithStringArg calls a C function with a char* arg and double return
func callFuncDoubleWithStringArg(fnPtr uintptr, strArg unsafe.Pointer) (float64, error) {
	return float64(C.call_fnd_s1(unsafe.Pointer(fnPtr), (*C.char)(strArg))), nil
}

// callFuncDoubleWithIntArg calls a C function with a long arg and double return
func callFuncDoubleWithIntArg(fnPtr uintptr, intArg uintptr) (float64, error) {
	return float64(C.call_fnd_i1(unsafe.Pointer(fnPtr), C.long(intArg))), nil
}

// callFuncWithStrings calls a C function passing char* pointers as uintptr args.
// On 64-bit platforms, sizeof(char*) == sizeof(long), so this works through the int convention.
func callFuncWithStrings(fnPtr uintptr, strArgs []unsafe.Pointer, intArgs []uintptr) (uintptr, error) {
	allArgs := make([]uintptr, len(strArgs)+len(intArgs))
	for i, p := range strArgs {
		allArgs[i] = uintptr(p)
	}
	for i, v := range intArgs {
		allArgs[len(strArgs)+i] = v
	}
	return callFunc(fnPtr, allArgs)
}

func callFunc(fnPtr uintptr, args []uintptr) (uintptr, error) {
	fn := unsafe.Pointer(fnPtr)
	switch len(args) {
	case 0:
		return uintptr(C.call_fn0(fn)), nil
	case 1:
		return uintptr(C.call_fn1(fn, C.long(args[0]))), nil
	case 2:
		return uintptr(C.call_fn2(fn, C.long(args[0]), C.long(args[1]))), nil
	case 3:
		return uintptr(C.call_fn3(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]))), nil
	case 4:
		return uintptr(C.call_fn4(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]), C.long(args[3]))), nil
	case 5:
		return uintptr(C.call_fn5(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]), C.long(args[3]), C.long(args[4]))), nil
	case 6:
		return uintptr(C.call_fn6(fn, C.long(args[0]), C.long(args[1]), C.long(args[2]), C.long(args[3]), C.long(args[4]), C.long(args[5]))), nil
	default:
		return 0, errors.New("cgo/call: too many arguments (max 6)")
	}
}
