; Hello World example
(println "Hello, World!")

; Variables
(define name "LL")
(println "Welcome to" name)

; String operations
(println "Uppercase:" (string-upcase "hello from ll!"))
(println "Length:" (string-length "hello"))

; Simple math
(define a 10)
(define b 20)
(println "a + b =" (+ a b))
(println "a * b =" (* a b))

; Conditionals
(if (> a b)
  (println "a is bigger")
  (println "b is bigger"))

; Functions
(define (square x) (* x x))
(println "5^2 =" (square 5))

; Lambda
(define double (lambda (x) (* x 2)))
(println "Double 21 =" (double 21))

; List operations
(define nums '(1 2 3 4 5))
(println "Sum:" (foldl + 0 nums))
(println "Doubled:" (map (lambda (x) (* x 2)) nums))
(println "Even numbers:" (filter (lambda (x) (even? x)) '(1 2 3 4 5 6)))
