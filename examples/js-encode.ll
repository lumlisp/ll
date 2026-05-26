;; JS Encode example - transpile LL code to JavaScript
(define ll-code "
(define (fact n)
  (if (<= n 1)
      1
      (* n (fact (- n 1)))))

(define (fib n)
  (if (<= n 1)
      n
      (+ (fib (- n 1)) (fib (- n 2)))))

(display (fact 5))
(display (fib 10))
")

(println "Transpiled JavaScript:")
(println (js/encode-string ll-code))

;; Can also encode from a file
(println "\n--- Or from a file ---")
(println "Use (js/encode-file \"path/to/file.ll\")")
