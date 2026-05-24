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
(define-macro (import module-name)
  (define (find-module-file name paths)
    (cond
      ((null? paths) #f)
      ((module-exists? (car paths) name)
       (string-append (car paths) "/" name ".ll"))
      (#t (find-module-file name (cdr paths)))))
  
  (define found-path (find-module-file module-name *module-paths*))
  
  (if found-path
      (list 'require found-path)  ; found-path - это строка во время компиляции
      (list 'println "ERROR: Module not found:" module-name)))



(add-module-path "/etc/ll/modules")
(add-module-path "ll_modules")
