;; WebSocket example - echo server
;; Run this file, then in another terminal connect with:
;;   websocat ws://localhost:9999

(define s (ws/create-server "localhost" 9999))
(ws/set-handler s (lambda (conn msg)
  (println "Received:" msg)
  (ws/send conn (string-append "echo: " msg))))

(println "Starting WebSocket echo server on ws://localhost:9999")
(define server-future (ws/start-server s))

;; Keep the REPL alive by awaiting the server future
(await server-future)
