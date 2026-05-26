;; CGO FFI example - load shared libraries
;; NOTE: Requires CGO_ENABLED=1 to build

(println "CGO FFI example")
(println "To use: (cgo/open \"libmylib.so\")")
(println "        (cgo/func lib \"my_function\")")
(println "        (cgo/call lib \"my_function\" arg1 arg2)")

(println "\nExample with math library (libm.so on Linux):")
(println "--- Integer functions (pass args as integers) ---")
(define lib (cgo/open "libm.so.6"))

;; abs is a pure integer function (takes int, returns int)
(cgo/func lib "abs")
(define result-int (cgo/call lib "abs" -42))
(println "abs(-42) =" result-int)

(println "\n--- Float functions (pass at least one arg as float) ---")
(cgo/func lib "sqrt")
(define result-f1 (cgo/call lib "sqrt" 144.0))
(println "sqrt(144.0) =" result-f1)

(define result-f2 (cgo/call lib "sqrt" 2.0))
(println "sqrt(2.0) =" result-f2)

(cgo/func lib "sin")
(define result-sin (cgo/call lib "sin" 0.0))
(println "sin(0.0) =" result-sin)

(cgo/close lib)
(println "\nDone!")
