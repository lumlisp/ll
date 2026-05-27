;; JS Transpile example - demonstrates OOP, DOM, FS, and import transpilation
(println "=== OOP Transpilation ===")
(println (js/encode-string "
(defclass Shape () ((x 0) (y 0)))
(defmethod Shape area (self) 0)
(defclass Circle (Shape) ((radius 1)))
(define c (new Circle 'radius 5))
(. c area)
(send c 'area)
(slot-ref c 'x)
(slot-set! c 'x 10)
(instance? c)
(class-of c)
"))

(println "=== DOM Transpilation ===")
(println (js/encode-string "
(define el (dom/id \"main\"))
(dom/set-text! el \"hello\")
(dom/add-class! el 'active)
(dom/on el 'click (lambda () (println \"clicked\")))
"))

(println "=== FS Transpilation ===")
(println (js/encode-string "
(define content (file->string \"input.txt\"))
(string->file \"output.txt\" content)
(file-exists? \"test.txt\")
(delete-file \"tmp.txt\")
"))

(println "=== Import Fallback (js/encode-string) ===")
(println (js/encode-string "
(import \"my-module\")
"))
