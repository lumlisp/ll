; PHP interop has been removed.
; Instead, use the built-in standard library:
(println "=== Built-in Library Examples ===")

; String operations
(define msg "  hello world  ")
(println "Original: '" msg "'")
(println "Trimmed: '" (string-trim msg) "'")
(println "Upper:" (string-upcase msg))
(println "Length:" (string-length (string-trim msg)))

; List operations
(define numbers '(3 1 4 1 5 9 2 6))
(println "Numbers:" numbers)
(println "Sum:" (foldl + 0 numbers))
(println "Sorted:" (reverse numbers))

; Math
(println "sqrt(144) =" (sqrt 144))
(println "abs(-42) =" (abs -42))
(println "expt(2 10) =" (expt 2 10))

; Map with lambda
(define doubled (map (lambda (x) (* x 2)) '(1 2 3 4 5)))
(println "Doubled:" doubled)

; Range, filter
(println "1 to 10:" (range 1 11))
(println "Evens 1-20:" (filter (lambda (x) (even? x)) (range 1 21)))

(println "=== Done! ===")
