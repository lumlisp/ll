# Lum Lisp

A Lisp dialect implemented in Go.

```
go build -o ll .
./ll                      # REPL
./ll file.ll [args...]    # run file with arguments
./ll -h, --help           # usage
./ll -v, --version        # version info
./ll -b file.ll [-o out]  # bundle script + deps into executable
```

## Quick start

```sh
go build -o ll .
./ll examples/hello.ll
```

## CLI arguments

Script arguments are available via `*args*` variable:

```sh
./ll script.ll foo bar 42
```

```scheme
(println *args*)  ; => (foo bar 42)
(car *args*)      ; => foo
```

## Built-in functions

Full reference in [docs/reference.md](docs/reference.md). Highlights:

| Category | Functions |
|----------|-----------|
| Arithmetic | `+` `-` `*` `/` `%` `abs` `min` `max` `expt` `sqrt` `quotient` `remainder` `floor` `ceil` `round` `inc` `dec` |
| Comparisons | `=` `>` `<` `>=` `<=` |
| Lists | `car` `cdr` `cons` `list` `length` `append` `reverse` `list-ref` `list-tail` `take` `drop` `range` `member` `assoc` `map` `filter` `foldl` `foldr` |
| Vectors | `vector` `make-vector` `vector-ref` `vector-set!` `vector-length` `vector-fill!` `vector-map` `vector->list` `list->vector` |
| Strings | `string-length` `string-ref` `substring` `string-append` `string=?` `string-ci=?` `string<?` `string>?` `string-downcase` `string-upcase` `string-trim` `string-split` `string-join` |
| Conversion | `number->string` `string->number` `symbol->string` `string->symbol` |
| I/O | `display` `write` `print` `println` `newline` `read-line` |
| File | `file->string` `string->file` `file-exists?` `delete-file` |
| System | `system` `shell->string` `sleep` `usleep` `exit` `get-file-dir` |
| OOP | `make-class` `new` `send` `slot-ref` `slot-set!` `instance?` `class-of` `add-method` |
| Predicates | `null?` `pair?` `list?` `symbol?` `number?` `integer?` `float?` `string?` `boolean?` `fn?` `future?` `vector?` `zero?` `even?` `odd?` `positive?` `negative?` `not` `equal?` `eq?` |

## Module system

Modules are installed via `lpm` or placed manually in `ll_modules/`.

```
ll_modules/lib.ll           # import as (import "lib")
ll_modules/curl/main.ll     # import as (import "curl")
```

```scheme
(import "lib")
(import "curl")
```

### Search paths

1. `ll_modules/` (relative to CWD)
2. `/etc/ll/modules/`

Add custom paths at runtime:

```scheme
(add-module-path "/my/modules")
(remove-module-path "/my/modules")
```

## curl library

HTTP requests via `curl` CLI:

```scheme
(import "curl")

; GET
(define body (curl/get "https://api.example.com/data"))
(println (curl/status))        ; 200

; POST with data
(curl/post "https://httpbin.org/post" "key=value")

; PUT
(curl/put "https://httpbin.org/put" "{\"x\":1}")

; PATCH
(curl/patch "https://httpbin.org/patch" "x=2")

; DELETE
(curl/delete "https://httpbin.org/delete")

; HEAD (returns response headers)
(curl/head "https://httpbin.org/get")

; With custom headers
(curl/get-with-headers "https://api.example.com/secret"
  '("Authorization: Bearer token123"))

; Helper
(println (curl/status))            ; last HTTP status code
(println (curl/response-headers))  ; last response headers as JSON
```

## lpm — Package manager

```sh
./lpm init                        # create ll_modules/ + .gitignore
./lpm require vendor/package      # clone github.com/vendor/package.git → ll_modules/vendor/package
./lpm require --global vendor/pkg # → /etc/ll/modules/vendor/package
```

## Documentation

See [docs/reference.md](docs/reference.md) for full language reference.

## Testing

```sh
go test ./...      # unit tests
./test-a-ll        # full test suite (unit tests + examples)
```
