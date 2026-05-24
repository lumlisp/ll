; FizzBuzz
(define (fizzbuzz n)
  (for i 1 (+ n 1)
    (cond
      ((= 0 (% i 15)) (println "FizzBuzz"))
      ((= 0 (% i 5))  (println "Buzz"))
      ((= 0 (% i 3))  (println "Fizz"))
      (else           (println i)))))

(println "FizzBuzz 1-20:")
(fizzbuzz 20)
