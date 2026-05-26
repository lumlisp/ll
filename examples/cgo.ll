;; CGO FFI example - load shared libraries
;; NOTE: Requires CGO_ENABLED=1 to build

(println "CGO FFI example")
(println "To use: (cgo/open \"libmylib.so\")")
(println "        (cgo/func lib \"my_function\")")
(println "        (cgo/call lib \"my_function\" arg1 arg2)")

(println "\nExample with math library (libm.so on Linux):")
(define lib (cgo/open "libm.so.6"))
(cgo/func lib "sqrt")
(define result (cgo/call lib "sqrt" 144))
(println "sqrt(144) =" result)

(cgo/close lib)
