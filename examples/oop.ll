;; Object-Oriented Programming in Lum Lisp
;;
;; Demonstrates: classes, inheritance, polymorphism,
;; method overriding, encapsulation (_ prefix convention),
;; slots with defaults, chaining methods

;; --- Base class ---

(defclass Shape () ((x 0) (y 0)))

(defmethod Shape area (self)
  0)

(defmethod Shape describe (self)
  (println (string-append "Shape at ("
    (number->string ($ x)) "," (number->string ($ y))
    ") area=" (number->string (. self area)))))

(defmethod Shape move (self dx dy)
  ($= x (+ ($ x) dx))
  ($= y (+ ($ y) dy))
  self)  ;; return self for chaining

(defmethod Shape distance-to (self other)
  (define dx (- (slot-ref other 'x) ($ x)))
  (define dy (- (slot-ref other 'y) ($ y)))
  (sqrt (+ (* dx dx) (* dy dy))))

;; "Private" method by convention (starts with _)
(defmethod Shape _validate (self)
  #t)

;; --- Rectangle ---

(defclass Rectangle (Shape) ((width 1) (height 1)))

(defmethod Rectangle area (self)
  (* ($ width) ($ height)))

(defmethod Rectangle describe (self)
  (println (string-append "Rectangle " (number->string ($ width))
    "x" (number->string ($ height)) " at ("
    (number->string ($ x)) "," (number->string ($ y))
    ") area=" (number->string (. self area)))))

(defmethod Rectangle _validate (self)
  (and (> ($ width) 0) (> ($ height) 0)))

;; --- Circle ---

(defclass Circle (Shape) ((radius 1)))

(defmethod Circle area (self)
  (* 3.14159 ($ radius) ($ radius)))

(defmethod Circle describe (self)
  (println (string-append "Circle r=" (number->string ($ radius))
    " at (" (number->string ($ x)) "," (number->string ($ y))
    ") area=" (number->string (. self area)))))

(defmethod Circle _validate (self)
  (> ($ radius) 0))

;; --- Usage ---

(println "=== Creating shapes ===")

(define r (new Rectangle 'x 10 'y 20 'width 5 'height 3))
(. r describe)
(println "  valid?" (. r _validate))

(define c (new Circle 'x 0 'y 0 'radius 10))
(. c describe)
(println "  valid?" (. c _validate))

(println)
(println "=== Move + method chaining ===")
(. (. r move 1 2) describe)

(println)
(println "=== Polymorphism: iterate over shapes ===")
(define shapes (list r c))
(map (lambda (s) (. s describe)) shapes)

(println)
(println "=== Distance between shapes ===")
(println "distance:" (. r distance-to c))

(println)
(println "=== Invalid circle ===")
(define bad (new Circle 'radius 0))
(println "  valid?" (. bad _validate))
