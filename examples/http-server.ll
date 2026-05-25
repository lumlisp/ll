(define server (http/create-server "localhost" 9999))

(http/set-handler server (lambda (req)
  (define method (http/request-method req))
  (define path (http/request-path req))
  (define body (http/request-body req))
  (define headers (http/request-headers req))

  (println "Request: " method " " path)
  (println "Headers: " headers)
  (println "Body: " body)

  (http/make-response 200
    '(("Content-Type" . "text/plain"))
    (string-append "Hello from LL!\n"
                   "Method: " method "\n"
                   "Path: " path "\n"
                   "Body: " body "\n"))))

(println "Starting server on http://localhost:9999")
(http/start-server server)
