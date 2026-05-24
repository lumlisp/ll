; list-ports.ll — показывает занятые сетевые порты
; использует ss (socket statistics) из пакета iproute2

(println "")
(println "=== Занятые TCP порты (LISTEN) ===")
(println "Proto  Local Address          Port  PID/Program")
(println "------ --------------------- ------ ------------")

; ss -tlnp4 — TCP, LISTEN, numeric, процессы, IPv4
(define raw (shell->string "ss -tlnp4"))
(define lines (string-split raw "\n"))

(for i 1 (length lines)
  (define line (list-ref lines i))
  (if (> (string-length line) 0)
    (println line)))

(println "")
(println "=== Занятые UDP порты ===")
(println "Proto  Local Address          Port  PID/Program")
(println "------ --------------------- ------ ------------")

(define raw-udp (shell->string "ss -ulnp4"))
(define udp-lines (string-split raw-udp "\n"))

(for i 1 (length udp-lines)
  (define line (list-ref udp-lines i))
  (if (> (string-length line) 0)
    (println line)))

(println "")
(println "=== Всего портов в прослушке ===")
(system "ss -tlnp4 | tail -n +2 | wc -l")
