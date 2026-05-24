(define (fib n)
  (if (<= n 1)
    n
    (+ (fib (- n 1)) (fib (- n 2)))))


(for i 0 15
  (println "fib(" i ") = " (fib i)))

(define (dump var)
  (php:var_dump var)
  var)


(dump (fib 10))


(if (> (fib 10) 10)
  (println "fib(10) > 10")
  (println "fib(10) <= 10"))

(php:require "test.php")
(define test (php:new "test" 10))

(define test::property "Иди нахуй")
(set! test::num (+ test::num 1))
(dump fib)
