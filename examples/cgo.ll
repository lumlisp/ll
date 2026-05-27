;; CGO FFI example - load shared libraries
;; NOTE: Requires CGO_ENABLED=1 to build

(println "CGO FFI example")
(println "To use: (cgo/open \"libmylib.so\")")
(println "        (cgo/func lib \"my_function\")")
(println "        (cgo/call lib \"my_function\" arg1 arg2)")
(println "")
(println "With types: (cgo/func lib \"fn\" '(arg-types... ret-type))")
(println "  e.g. (cgo/func lib \"strlen\" '(string int))")

(println "\n=== Auto-detected (legacy) ===")
(define libm (cgo/open "libm.so.6"))
(cgo/func libm "abs")
(define r1 (cgo/call libm "abs" -42))
(println "abs(-42) =" r1)

(cgo/func libm "sqrt")
(define r2 (cgo/call libm "sqrt" 144.0))
(println "sqrt(144.0) =" r2)

(cgo/close libm)

(println "\n=== Typed FFI ===")
(define libc (cgo/open "libc.so.6"))

;; String arg, int return
(cgo/func libc "strlen" '(string int))
(define len (cgo/call libc "strlen" "Hello World!"))
(println "strlen(\"Hello World!\") =" len)

;; String arg, string return
(cgo/func libc "getenv" '(string string))
(define home (cgo/call libc "getenv" "HOME"))
(println "HOME =" home)

;; Explicit typed double
(cgo/func libc "atof" '(string double))
(define num (cgo/call libc "atof" "3.14159"))
(println "atof(\"3.14159\") =" num)

;; Explicit typed ints
(cgo/func libc "abs" '(int int))
(define r3 (cgo/call libc "abs" -99))
(println "abs(-99) =" r3)

(cgo/close libc)
(println "\nDone!")
