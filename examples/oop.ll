;; OOP example

(defclass Animal () ((name "unknown") (age 0)))

(defmethod Animal speak (self)
  (println ($ name) "says hello"))

(defmethod Animal describe (self)
  (println ($ name) "is" ($ age) "years old"))

(define a (new Animal 'name "Rex" 'age 5))
(. a describe)
(. a speak)

(defclass Dog (Animal) ((breed "mixed")))

(defmethod Dog speak (self)
  (println ($ name) "barks! Breed:" ($ breed)))

(define d (new Dog 'name "Max" 'breed "Husky"))
(. d describe)
(. d speak)

;; Method with arguments
(defmethod Dog fetch (self thing)
  (println ($ name) "fetches" thing))

(. d fetch "the ball")
