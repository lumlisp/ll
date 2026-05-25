; Async example - future, await, co

; Basic future and await
(define f (future (+ 1 2)))
(println "future? f:" (future? f))
(println "await f:" (await f))

; co - async lambda (returns a future when called)
(define slow-add (co (a b)
  (+ a b)))

(println "co add:" (await (slow-add 10 20)))

; Multiple futures in parallel
(define f1 (future (* 3 4)))
(define f2 (future (* 5 6)))
(println "f1:" (await f1))
(println "f2:" (await f2))

; co with multiple body expressions
(define compute (co ()
  (define x 100)
  (define y 200)
  (+ x y)))

(println "compute:" (await (compute)))

; Nesting future/await
(define nested (future (await (future (+ 1 2)))))
(println "nested:" (await nested))
