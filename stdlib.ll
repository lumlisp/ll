(define *module-paths* ())

(define (normalize-path path)
  (if (string=? (substring path (- (string-length path) 1) (string-length path)) "/")
      (substring path 0 (- (string-length path) 1))
      path))

(define (remove elem lst)
  (cond
    ((null? lst) ())
    ((equal? elem (car lst)) (cdr lst))
    (#t (cons (car lst) (remove elem (cdr lst))))))

(define-macro (add-module-path path)
  (list 'set! '*module-paths* 
        (list 'cons (list 'normalize-path path) 
              (list 'remove (list 'normalize-path path) '*module-paths*))))

(define-macro (remove-module-path path)
  (list 'set! '*module-paths* 
        (list 'remove (list 'normalize-path path) '*module-paths*)))

(define (module-exists? base-path module-name)
  (file-exists? (string-append base-path "/" module-name ".ll")))

;;; Этот макрос ищет файл во ВРЕМЯ КОМПИЛЯЦИИ
;;; Сначала ищет <path>/<name>.ll, затем <path>/<name>/main.ll
(define-macro (import module-name)
  (define (find-module-file name paths)
    (cond
      ((null? paths) #f)
      ((module-exists? (car paths) name)
       (string-append (car paths) "/" name ".ll"))
      (#t (if (file-exists? (string-append (car paths) "/" name "/main.ll"))
              (string-append (car paths) "/" name "/main.ll")
              (find-module-file name (cdr paths))))))
  
  (define found-path (find-module-file module-name *module-paths*))
  
  (if found-path
      (list 'require found-path)  ; found-path - это строка во время компиляции
      (list 'println "ERROR: Module not found:" module-name)))



(add-module-path "/etc/ll/modules")
(add-module-path "ll_modules")

;; --- OOP ---

(define (_process-slot slot)
  (if (null? (cdr slot))
      (list (car slot) ())
      (list (car slot) (car (cdr slot)))))

(define (_process-slots slots)
  (if (null? slots)
      ()
      (cons (_process-slot (car slots))
            (_process-slots (cdr slots)))))

(define-macro (defclass name parent slots)
  (list 'define name
    (list 'make-class (list 'quote name)
      (if (null? parent) () (car parent))
      (list 'quote (_process-slots slots)))))

(define-macro (defmethod class name params &rest body)
  (list 'add-method class (list 'quote name)
    (cons 'lambda (cons params body))))

(define-macro (. obj method &rest args)
  (cons 'send (cons obj (cons (list 'quote method) args))))

(define-macro ($ slot-name)
  (list 'slot-ref 'self (list 'quote slot-name)))

(define-macro ($= slot-name value)
  (list 'slot-set! 'self (list 'quote slot-name) value))
