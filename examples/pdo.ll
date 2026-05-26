;; PDO example - SQLite in-memory database
(define db (pdo/open "sqlite" "file::memory:?cache=shared"))

(println "Creating table...")
(pdo/exec db "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")

(println "Inserting users...")
(pdo/exec db "INSERT INTO users (name, email) VALUES (?, ?)" "Alice" "alice@example.com")
(pdo/exec db "INSERT INTO users (name, email) VALUES (?, ?)" "Bob" "bob@example.com")
(pdo/exec db "INSERT INTO users (name, email) VALUES (?, ?)" "Charlie" "charlie@example.com")

(println "\nAll users:")
(define rows (pdo/query db "SELECT * FROM users ORDER BY id"))
(println "  " rows)

(println "\nUser with id=2:")
(define result (pdo/query db "SELECT * FROM users WHERE id = ?" 2))
(println "  " (car result))

(println "\nParameterized query (id > 1):")
(define filtered (pdo/query db "SELECT name, email FROM users WHERE id > ?" 1))
(println "  " filtered)

(pdo/close db)
(println "\nDone.")
