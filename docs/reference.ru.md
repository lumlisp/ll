# Справочник языка Lum Lisp

## Обзор

Lum Lisp — это диалект Lisp. В нём реализованы лексическая область видимости,
замыкания первого класса, векторы, макросы и обширная стандартная библиотека.

## Запуск

```sh
make build
./ll                   # REPL
./ll file.ll [args...] # запуск файла с аргументами
./ll -h, --help        # справка
./ll -v, --version     # версия
./ll -b file.ll [-o out]  # упаковка скрипта + зависимостей в исполняемый файл
```

Аргументы скрипта доступны через переменную `*args*` (список строк).

## Синтаксис

### Комментарии
```
; строчный комментарий
```

### Shebang
```
#!/usr/bin/env ll
```
Распознаётся только на самой первой строке файла.

### Литералы

| Синтаксис  | Тип       |
|------------|-----------|
| `42`       | Integer   |
| `-7`       | Integer   |
| `3.14`     | Float     |
| `"hello"`  | String    |
| `#t`       | Boolean true  |
| `#f`       | Boolean false |
| `'x`       | Сокращение от `(quote x)` |
| `(a b c)`  | Список (ячейки Cons) |
| `(a . b)`  | Точечная пара (несокращённый список) |
| `(a b . c)` | Точечная пара с хвостом |
| `#(1 2 3)` | Вектор |

### Идентификаторы (Symbols)
```
foo  bar?  +  <=>  my-func_1
```
Почти любой символ, кроме пробелов, `(`, `)`, `"`, `;`, `#` (если не начало `#t`/`#f`/`#(`).

## Типы

| Тип         | Представление     | Truthy? |
|-------------|-------------------|---------|
| `Integer`   | 64-битное целое   | Всегда  |
| `Float`     | 64-битное с плав. | Всегда  |
| `String`    | UTF-8 текст       | Всегда  |
| `Boolean`   | `#t` / `#f`       | Как есть |
| `Symbol`    | Именованный идентификатор | Всегда |
| `Cons`      | Пара `(a . b)` или список `(a b c)` | Всегда |
| `Nil`       | `()` (пустой список) | **False** |
| `Vector`    | `#(1 2 3)`        | Всегда  |
| `Closure`   | Пользовательская функция | Всегда |
| `Primitive` | Встроенная функция | Всегда |
| `Macro`     | Макрос (define-macro) | Всегда |
| `Future`    | Асинхронный результат (future/co) | Всегда |

Только `#f` и `()` являются ложными; всё остальное истинно.

## Специальные формы

### `define`
```
(define x 42)
(define (fn a b) (+ a b))
(define (fn a &rest rest) ...)
```
`&rest` захватывает оставшиеся аргументы как список.

### `set!`
```
(set! x 99)
```

### `if`
```
(if cond then-expr else-expr)
```

### `cond`
```
(cond
  (test1 expr1)
  (test2 expr2)
  (else expr3))
```

### `lambda`
```
(lambda (x y) (+ x y))
(lambda (x &rest rest) (apply + x rest))
```

### `quote`
```
(quote (1 2 3))   ; => (1 2 3)
```

### `begin`
```
(begin expr1 expr2 expr3)   ; возвращает последнее
```

### `while`
```
(while condition body ...)
```

### `for`
```
(for var start end body ...)
```
Итерирует `var` от `start` до `end` (не включая), увеличивая на 1 на каждом шаге.

### `and` / `or`
```
(and expr ...)    ; короткое замыкание
(or expr ...)     ; короткое замыкание
```

### `require` / `include`
```
(require "file.ll")   ; загружает один раз (отслеживает загруженные файлы)
(includes "file.ll")   ; загружает каждый раз
```
Пути указываются относительно директории текущего файла. Также поддерживаются модули-директории: `(require "mymod")` ищет `mymod/main.ll`, если путь — директория.

### `future`
```
(future expr ...)
```
Вычисляет выражения тела конкурентно и возвращает значение `Future`.
Используйте `await` для получения результата.

```scheme
(define f (future (+ 1 2)))
(println (await f))  ; => 3
```

### `await`
```
(await future)
```
Блокируется до разрешения `Future`, затем возвращает его значение.
Если вычисление future вызвало ошибку, `await` пробрасывает её.

```scheme
(await (future (* 3 4)))  ; => 12
```

### `co`
```
(co (params ...) body ...)
```
Создаёт **асинхронное замыкание** — как `lambda`, но вызов выполняется конкурентно
и возвращает `Future` вместо синхронного выполнения.

```scheme
(define slow-add (co (a b)
  (+ a b)))

(println (await (slow-add 10 20)))  ; => 30
```

Используйте с `define` для создания именованных асинхронных функций. Несколько future
могут выполняться параллельно:

```scheme
(define f1 (slow-add 1 2))
(define f2 (slow-add 3 4))
(println (await f1))  ; => 3
(println (await f2))  ; => 7
```

### `define-macro`
```
(define-macro (name params ...) body)
(define-macro (unless cond body) (list (quote if) (list (quote not) cond) body))
```
Макросы получают невычисленные выражения аргументов и возвращают выражение для вычисления.

## Встроенные переменные

| Переменная | Значение |
|------------|---------|
| `*args*` | Список аргументов командной строки, переданных скрипту (пуст в REPL) |

## Встроенные функции

| Функция | Описание |
|---------|----------|
| `(get-file-dir)` | Возвращает абсолютную директорию текущего исполняемого файла (как PHP `__DIR__`), или `""` в REPL |

## Система модулей

Модули загружаются через макрос `import`, который разрешает имена модулей на этапе компиляции по `*module-paths*`.

```
(add-module-path "/path/to/modules")   ; добавить путь поиска
(remove-module-path "/path/to/modules") ; удалить путь поиска
(import "lib")                          ; загрузить <path>/lib.ll из первого совпадения
(import "curl")                         ; или загрузить <path>/curl/main.ll, если директория
```

Пути поиска по умолчанию:

- `/etc/ll/modules`
- `ll_modules` (относительно текущей директории)

Порядок разрешения модуля для каждого пути:

1. `<path>/<name>.ll` — обычный файл
2. `<path>/<name>/main.ll` — модуль-директория (как Node.js `index.js`)

## Стандартная библиотека

### Арифметика
| Функция | Описание |
|---------|----------|
| `(+ a ...)` | Сумма |
| `(- a ...)` | Вычитание |
| `(* a ...)` | Умножение |
| `(/ a ...)` | Деление |
| `(% a b)`   | Остаток от деления |
| `(abs n)`   | Абсолютное значение |
| `(min a b ...)` | Минимум |
| `(max a b ...)` | Максимум |
| `(expt base pow)` | Возведение в степень |
| `(sqrt n)`  | Квадратный корень |
| `(quotient a b)` | Целочисленное деление |
| `(remainder a b)` | Остаток |
| `(floor n)` | Округление вниз |
| `(ceil n)`  | Округление вверх |
| `(round n)` | Округление до ближайшего |
| `(inc n)`   | `(+ n 1)` |
| `(dec n)`   | `(- n 1)` |

### Сравнения
| Функция | Возвращает |
|---------|------------|
| `(= a b)` | `#t` если a и b численно равны |
| `(> a b)` | `#t` если a > b |
| `(< a b)` | `#t` если a < b |
| `(>= a b)` | `#t` если a >= b |
| `(<= a b)` | `#t` если a <= b |

### Операции со списками
| Функция | Описание |
|---------|----------|
| `(car pair)` | Первый элемент |
| `(cdr pair)` | Остаток |
| `(cons a b)` | Создать пару |
| `(list a ...)` | Создать список |
| `(null? x)`   | `#t` если Nil |
| `(pair? x)`   | `#t` если Cons |
| `(list? x)`   | `#t` если правильный список |
| `(length lst)` | Длина списка |
| `(append lst ...)` | Склеить списки |
| `(reverse lst)` | Развернуть список |
| `(list-ref lst n)` | N-й элемент (с 0) |
| `(list-tail lst n)` | N-й cdr |
| `(take lst n)` | Первые n элементов |
| `(drop lst n)` | Все кроме первых n |
| `(range end)` | Целые от 0 до end-1 |
| `(range start end)` | Целые от start до end-1 |
| `(member x lst)` | Первый хвост начинающийся с x, или `#f` |
| `(assoc key alist)` | Поиск ключа в ассоциативном списке |
| `(map fn lst)` | Применить fn к каждому элементу |
| `(filter pred lst)` | Оставить элементы, удовлетворяющие pred |
| `(foldl fn init lst)` | Левая свёртка |
| `(foldr fn init lst)` | Правая свёртка |

### Предикаты
| Функция | Описание |
|---------|----------|
| `(symbol? x)` | `#t` если Symbol |
| `(number? x)` | `#t` если Integer или Float |
| `(integer? x)` | `#t` если Integer |
| `(float? x)` | `#t` если Float |
| `(string? x)` | `#t` если String |
| `(boolean? x)` | `#t` если Boolean |
| `(fn? x)` | `#t` если Closure или Primitive |
| `(future? x)` | `#t` если Future |
| `(zero? n)` | `#t` если 0 |
| `(even? n)` | `#t` если чётное |
| `(odd? n)` | `#t` если нечётное |
| `(positive? n)` | `#t` если > 0 |
| `(negative? n)` | `#t` если < 0 |
| `(not x)` | Отрицание |
| `(equal? a b)` | Структурное равенство |
| `(eq? a b)` | Синоним для `equal?` |

### Строковые операции
| Функция | Описание |
|---------|----------|
| `(string-length s)` | Количество символов |
| `(string-ref s n)` | N-й символ (как строка) |
| `(substring s start end)` | Срез |
| `(string-append s ...)` | Склеивание |
| `(string=? a b)` | Чувствительное к регистру равенство |
| `(string-ci=? a b)` | Нечувствительное к регистру равенство |
| `(string<? a b)` | Меньше |
| `(string>? a b)` | Больше |
| `(string-downcase s)` | Нижний регистр |
| `(string-upcase s)` | Верхний регистр |
| `(string-trim s)` | Удаление пробелов по краям |
| `(string-split s)` | Разделение по пробелам |
| `(string-split s sep)` | Разделение по разделителю |
| `(string-join parts sep)` | Склеивание с разделителем |
| `(number->string n)` | Число в строку |
| `(string->number s)` | Строку в число (или `#f`) |
| `(symbol->string s)` | Имя символа в строку |
| `(string->symbol s)` | Строку в символ |

### Векторы
| Функция | Описание |
|---------|----------|
| `(vector x ...)` | Создать вектор |
| `(make-vector n)` | Вектор длины n, заполненный `()` |
| `(make-vector n fill)` | Вектор длины n, заполненный `fill` |
| `(vector-ref v i)` | Индекс (с 0) |
| `(vector-set! v i x)` | Изменить элемент |
| `(vector-length v)` | Длина |
| `(vector? x)` | `#t` если Vector |
| `(vector->list v)` | Преобразовать в список |
| `(list->vector lst)` | Преобразовать в вектор |
| `(vector-fill! v x)` | Заполнить все элементы |
| `(vector-map fn v)` | Применить fn к элементам |

### Ввод/Вывод
| Функция | Описание |
|---------|----------|
| `(display x)` | Печать без новой строки (без кавычек) |
| `(print x ...)` | Печать через пробел, без новой строки |
| `(println x ...)` | Печать через пробел и с новой строкой |
| `(newline)` | Печать новой строки |
| `(write x)` | Печать с кавычками |
| `(read-line)` | Чтение строки из stdin (возвращает строку или `()` на EOF) |
| `(file->string path)` | Чтение файла в строку |
| `(string->file path content)` | Запись строки в файл |
| `(file-exists? path)` | `#t` если файл существует |
| `(delete-file path)` | Удалить файл |

### Объектно-ориентированное программирование
| Функция | Описание |
|---------|----------|
| `(make-class name parent slots-defs)` | Создать класс (сырая встроенная) |
| `(new class key val ...)` | Создать экземпляр со значениями слотов |
| `(send obj method-name args...)` | Вызвать метод на экземпляре |
| `(slot-ref obj slot-name)` | Прочитать значение слота |
| `(slot-set! obj slot-name val)` | Записать значение слота |
| `(instance? x)` | `#t` если Instance |
| `(class-of x)` | Вернуть класс экземпляра |
| `(add-method class name fn)` | Добавить метод в класс |

### Макросы (удобный слой)

| Макрос | Описание |
|--------|----------|
| `(defclass Name (Parent) ((slot default)...))` | Определить класс с опциональным наследованием |
| `(defmethod Class method-name (self ...) body...)` | Определить метод (первый параметр — `self`) |
| `(. obj method args...)` | Синтаксический сахар для вызова метода |
| `($ slot-name)` | Доступ к слоту внутри методов (использует `self`) |
| `($= slot-name value)` | Изменение слота внутри методов |

```scheme
;; --- Пример ---

(defclass Animal () ((name "unknown") (age 0)))

(defmethod Animal speak (self)
  (println ($ name) "says hello"))

(define a (new Animal 'name "Rex"))
(. a speak)                       ; => Rex says hello

;; Наследование
(defclass Dog (Animal) ((breed "mixed")))

(defmethod Dog speak (self)
  (println ($ name) "barks! Breed:" ($ breed)))

(define d (new Dog 'name "Max" 'breed "Husky"))
(. d speak)                       ; => Max barks! Breed: Husky
(. d make-older 2)                ; унаследованный метод
```

### HTTP-сервер

Встроенный HTTP-сервер на основе стандартной HTTP-библиотеки. Обработчики получают объект запроса
и должны вернуть ответ, созданный с помощью `http/make-response`.

| Функция | Описание |
|---------|----------|
| `(http/create-server host port)` | Создать HTTP-сервер (возвращает объект сервера) |
| `(http/set-handler server handler)` | Установить обработчик запросов (функция с одним аргументом — запросом) |
| `(http/start-server server)` | Запустить сервер (блокирующий вызов) |
| `(http/request-method req)` | Вернуть метод HTTP (GET, POST, и т.д.) |
| `(http/request-path req)` | Вернуть путь запроса |
| `(http/request-headers req)` | Вернуть заголовки как ассоциативный список `((ключ . значение) ...)` |
| `(http/request-body req)` | Вернуть тело запроса как строку |
| `(http/make-response status headers body)` | Создать HTTP-ответ |
| `(http/response-status resp)` | Вернуть код статуса ответа |
| `(http/response-headers resp)` | Вернуть заголовки ответа как ассоциативный список `((ключ . значение) ...)` |
| `(http/response-body resp)` | Вернуть тело ответа |

```scheme
;; Простой hello-сервер
(define server (http/create-server "localhost" 8080))

(http/set-handler server (lambda (req)
  (define method (http/request-method req))
  (define path (http/request-path req))
  (define body (http/request-body req))

  (http/make-response 200
    '(("Content-Type" . "text/plain"))
    (string-append "Method: " method "\n"
                   "Path: " path "\n"
                   "Body: " body "\n"))))

(println "Listening on http://localhost:8080")
(http/start-server server)
```

```scheme
;; Пример JSON-эндпоинта
(define server (http/create-server "0.0.0.0" 3000))

(http/set-handler server (lambda (req)
  (if (string=? (http/request-path req) "/api/hello")
    (http/make-response 200
      '(("Content-Type" . "application/json"))
      "{\"message\": \"Hello from LL!\"}")
    (http/make-response 404
      '(("Content-Type" . "text/plain"))
      "Not Found"))))

(http/start-server server)
```

### Системные
| Функция | Описание |
|---------|----------|
| `(system cmd)` | Выполнить команду, вывести вывод, вернуть код возврата |
| `(shell->string cmd)` | Выполнить команду, захватить stdout как строку |
| `(sleep n)` | Спать n секунд (целое или с плавающей точкой) |
| `(usleep n)` | Спать n миллисекунд |
| `(exit)` | Выйти с кодом 0 |
| `(exit n)` | Выйти с кодом n |

### JSON

| Функция | Описание |
|---------|----------|
| `(json/encode val)` | Преобразует значение LL в JSON-строку |
| `(json/decode str)` | Разбирает JSON-строку в значение LL |

```scheme
(json/encode '(1 2 3))                      ; => "[1,2,3]"
(json/encode '((name . "LL") (year . 2024))) ; => "{\"name\":\"LL\",\"year\":2024}"
(json/decode "{\"x\":1,\"y\":2}")            ; => ((x . 1) (y . 2))
```

### PDO — База данных

| Функция | Описание |
|---------|----------|
| `(pdo/open dsn user password)` | Открывает соединение с БД (возвращает объект соединения или `()` при ошибке) |
| `(pdo/exec conn sql . params)` | Выполняет INSERT/UPDATE/DELETE, возвращает количество затронутых строк |
| `(pdo/query conn sql . params)` | Выполняет SELECT, возвращает строки как список ассоциативных списков `((колонка . значение) ...)` |
| `(pdo/close conn)` | Закрывает соединение |

Поддерживаемые префиксы DSN:
- `sqlite:filename.db` — SQLite (встраиваемый)
- `mysql:user:pass@tcp(host:port)/dbname` — MySQL
- `postgres:host=... dbname=...` — PostgreSQL

Параметризованные запросы через `?`:

```scheme
(define db (pdo/open "sqlite:test.db" "" ""))
(pdo/exec db "CREATE TABLE IF NOT EXISTS users (id INTEGER, name TEXT)")
(pdo/exec db "INSERT INTO users (id, name) VALUES (?, ?)" 1 "Alice")
(define rows (pdo/query db "SELECT * FROM users WHERE id = ?" 1))
(pdo/close db)
```

### WebSocket

| Функция | Описание |
|---------|----------|
| `(ws/create-server host port)` | Создаёт WebSocket-сервер (возвращает объект сервера) |
| `(ws/set-handler server handler)` | Устанавливает обработчик подключений (функция с одним аргументом — соединением) |
| `(ws/start-server server)` | Запускает сервер (возвращает Future, не блокирует) |
| `(ws/connect url)` | Подключается к WebSocket-серверу (возвращает соединение или `()` при ошибке) |
| `(ws/send conn msg)` | Отправляет текстовое сообщение |
| `(ws/receive conn)` | Получает текстовое сообщение (блокирует) |
| `(ws/close conn)` | Закрывает соединение |

```scheme
;; Сервер
(define server (ws/create-server "localhost" 8080))
(ws/set-handler server (lambda (conn)
  (define msg (ws/receive conn))
  (ws/send conn (string-append "echo: " msg))))
(ws/start-server server)

;; Клиент
(define conn (ws/connect "ws://localhost:8080"))
(ws/send conn "hello")
(println (ws/receive conn))  ; => "echo: hello"
(ws/close conn)
```

### js/encode — Транспилятор LL → JavaScript

| Функция | Описание |
|---------|----------|
| `(js/encode-string expr)` | Преобразует выражение LL в JavaScript-строку |
| `(js/encode-file path expr)` | Преобразует выражение LL в JavaScript и записывает в файл |

Поддерживаемые формы LL: `define`, `lambda`, `if`, `cond`, `begin`, `set!`, арифметика, сравнения, строки, векторы, async (`future`, `co`, `await`), операции со списками (`car`, `cdr`, `cons`, `list`), `while`, `for`, `display`, `println`.

```scheme
(js/encode-string '(define (fib n)
  (if (< n 2) n
    (+ (fib (- n 1)) (fib (- n 2))))))
;; => "function fib(n) { if (n < 2) { return n; } else { return fib(n - 1) + fib(n - 2); } }"
```

### CGO/FFI — Привязка C-библиотек

| Функция | Описание |
|---------|----------|
| `(cgo/open path)` | Загружает shared library (возвращает объект библиотеки или `()` при ошибке) |
| `(cgo/func lib name)` | Находит функцию в библиотеке (регистрирует для вызова) |
| `(cgo/call lib name args...)` | Вызывает зарегистрированную функцию (до 6 аргументов) |
| `(cgo/close lib)` | Выгружает библиотеку |

Требует `CGO_ENABLED=1` при сборке. При `CGO_ENABLED=0` все функции возвращают ошибку.

**Определение соглашения о вызове:**
- Если **хотя бы один** аргумент — число с плавающей точкой (`3.14`, `144.0`), ВСЕ аргументы передаются как C `double` и результат возвращается как Float
- Если **все** аргументы — целые числа, они передаются как C `long` и результат возвращается как Integer
- Смешанные int/float в одном вызове работают (целые повышаются до double)
- Поддерживаются только числовые аргументы и указатели — нет строк, структур или void-функций

```scheme
(define lib (cgo/open "libm.so.6"))

;; Целочисленная функция: все аргументы — целые
(cgo/func lib "abs")
(println (cgo/call lib "abs" -42))   ; => 42

;; Функция с плавающей точкой: передайте хотя бы один float
(cgo/func lib "sqrt")
(println (cgo/call lib "sqrt" 144.0))  ; => 12.0

(cgo/func lib "sin")
(println (cgo/call lib "sin" 0.0))     ; => 0.0

(cgo/close lib)
```

## Примеры

Смотрите директорию `examples/`:
- `hello.ll` — Hello world, переменные, математика, списки
- `fib.ll` — Рекурсивный Фибоначчи
- `fizzbuzz.ll` — FizzBuzz с `for` и `cond`
- `php-interop.ll` — Демо стандартной библиотеки (замена старого PHP interop)
- `list-ports.ll` — Вывод системной команды через `shell->string`
- `async.ll` — `future`, `await`, `co` асинхронное программирование
- `http-server.ll` — Пример HTTP-сервера
- `pdo.ll` — Пример PDO (SQLite)
- `ws.ll` — Пример WebSocket сервера и клиента
- `js-encode.ll` — Пример транспилятора LL→JS
- `cgo.ll` — Пример CGO FFI (libm)
