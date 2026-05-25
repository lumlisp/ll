# Lum Lisp Language Reference

## Overview

Lum Lisp is a Lisp dialect implemented in Go. It features lexical scoping,
first-class closures, vectors, macros, and a comprehensive standard library.

## Running

```sh
go build -o ll .
./ll                   # REPL
./ll file.ll [args...] # run file with arguments
./ll -h, --help        # usage info
./ll -v, --version     # version info
./ll -b file.ll [-o out]  # bundle script + deps into executable
```

Script arguments are accessible via `*args*` variable (list of strings).

## Syntax

### Comments
```
; line comment
```

### Shebang
```
#!/usr/bin/env ll
```
Only recognized on the very first line of a file.

### Literals

| Syntax     | Type    |
|------------|---------|
| `42`       | Integer |
| `-7`       | Integer |
| `3.14`     | Float   |
| `"hello"`  | String  |
| `#t`       | Boolean true  |
| `#f`       | Boolean false |
| `'x`       | Shorthand for `(quote x)` |
| `(a b c)`  | List (Cons cells) |
| `#(1 2 3)` | Vector |

### Identifiers (Symbols)
```
foo  bar?  +  <=>  my-func_1
```
Almost any character except whitespace, `(`, `)`, `"`, `;`, `#` (unless starting `#t`/`#f`/`#(`).

## Types

| Type      | Representation | Truthy? |
|-----------|---------------|---------|
| `Integer` | 64-bit int    | Always  |
| `Float`   | 64-bit float  | Always  |
| `String`  | UTF-8 text    | Always  |
| `Boolean` | `#t` / `#f`   | As-is   |
| `Symbol`  | Named identifier | Always |
| `Cons`    | Pair `(a . b)` or list `(a b c)` | Always |
| `Nil`     | `()` (empty list) | **False** |
| `Vector`  | `#(1 2 3)`   | Always  |
| `Closure` | User-defined function | Always |
| `Primitive` | Built-in function | Always |
| `Macro`   | Macro (define-macro) | Always |
| `Future`  | Async result (future/co) | Always |

Only `#f` and `()` are falsey; everything else is truthy.

## Special Forms

### `define`
```
(define x 42)
(define (fn a b) (+ a b))
(define (fn a &rest rest) ...)
```
`&rest` captures remaining arguments as a list.

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
(begin expr1 expr2 expr3)   ; returns last
```

### `while`
```
(while condition body ...)
```

### `for`
```
(for var start end body ...)
```
Iterates `var` from `start` to `end` (exclusive), incrementing by 1 each step.

### `and` / `or`
```
(and expr ...)    ; short-circuit
(or expr ...)     ; short-circuit
```

### `require` / `include`
```
(require "file.ll")   ; loads once (tracks loaded files)
(include "file.ll")   ; loads every time
```
Paths are relative to the current file's directory. Also supports directory modules: `(require "mymod")` resolves to `mymod/main.ll` if the path is a directory.

### `future`
```
(future expr ...)
```
Evaluates body expressions in a new goroutine and returns a `Future` value.
Use `await` to retrieve the result.

```scheme
(define f (future (+ 1 2)))
(println (await f))  ; => 3
```

### `await`
```
(await future)
```
Blocks until the given `Future` resolves, then returns its value. If the future
computation raised an error, `await` re-throws it.

```scheme
(await (future (* 3 4)))  ; => 12
```

### `co`
```
(co (params ...) body ...)
```
Creates an **async closure** — like `lambda`, but calling it spawns a goroutine
and returns a `Future` instead of running synchronously.

```scheme
(define slow-add (co (a b)
  (+ a b)))

(println (await (slow-add 10 20)))  ; => 30
```

Use with `define` to create named async functions. Multiple futures can run in
parallel and be awaited later:

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
Macros receive unevaluated argument expressions and return an expression to evaluate.

## Built-in Variables

| Variable | Value |
|----------|-------|
| `*args*` | List of command-line arguments passed to the script (empty in REPL) |

## Built-in Functions

| Function | Description |
|----------|-------------|
| `(get-file-dir)` | Returns the absolute directory of the currently executing file (like PHP's `__DIR__`), or `""` in REPL |

## Module System

Modules are loaded via the `import` macro, which resolves module names at compile time against `*module-paths*`.

```
(add-module-path "/path/to/modules")   ; add search path
(remove-module-path "/path/to/modules") ; remove search path
(import "lib")                          ; load <path>/lib.ll from first match
(import "curl")                         ; or load <path>/curl/main.ll if directory
```

The default search paths are:

- `/etc/ll/modules`
- `ll_modules` (relative to current directory)

Module resolution order for each path:

1. `<path>/<name>.ll` — plain file
2. `<path>/<name>/main.ll` — directory module (like Node.js `index.js`)

## Standard Library

### Arithmetic
| Function | Description |
|----------|-------------|
| `(+ a ...)` | Sum |
| `(- a ...)` | Subtract |
| `(* a ...)` | Multiply |
| `(/ a ...)` | Divide |
| `(% a b)`   | Modulo |
| `(abs n)`   | Absolute value |
| `(min a b ...)` | Minimum |
| `(max a b ...)` | Maximum |
| `(expt base pow)` | Exponentiation |
| `(sqrt n)`  | Square root |
| `(quotient a b)` | Integer division |
| `(remainder a b)` | Remainder |
| `(floor n)` | Round down |
| `(ceil n)`  | Round up |
| `(round n)` | Round to nearest |
| `(inc n)`   | `(+ n 1)` |
| `(dec n)`   | `(- n 1)` |

### Comparisons
| Function | Returns |
|----------|---------|
| `(= a b)` | `#t` if a and b are numerically equal |
| `(> a b)` | `#t` if a > b |
| `(< a b)` | `#t` if a < b |
| `(>= a b)` | `#t` if a >= b |
| `(<= a b)` | `#t` if a <= b |

### List Operations
| Function | Description |
|----------|-------------|
| `(car pair)` | First element |
| `(cdr pair)` | Rest |
| `(cons a b)` | Construct pair |
| `(list a ...)` | Create list |
| `(null? x)`   | `#t` if Nil |
| `(pair? x)`   | `#t` if Cons |
| `(list? x)`   | `#t` if proper list |
| `(length lst)` | List length |
| `(append lst ...)` | Concatenate lists |
| `(reverse lst)` | Reverse list |
| `(list-ref lst n)` | Nth element (0-based) |
| `(list-tail lst n)` | Nth cdr |
| `(take lst n)` | First n elements |
| `(drop lst n)` | All but first n |
| `(range end)` | Integers from 0 to end-1 |
| `(range start end)` | Integers from start to end-1 |
| `(member x lst)` | First tail starting with x, or `#f` |
| `(assoc key alist)` | Lookup key in association list |
| `(map fn lst)` | Apply fn to each element |
| `(filter pred lst)` | Keep elements matching pred |
| `(foldl fn init lst)` | Left fold |
| `(foldr fn init lst)` | Right fold |

### Predicates
| Function | Description |
|----------|-------------|
| `(symbol? x)` | `#t` if Symbol |
| `(number? x)` | `#t` if Integer or Float |
| `(integer? x)` | `#t` if Integer |
| `(float? x)` | `#t` if Float |
| `(string? x)` | `#t` if String |
| `(boolean? x)` | `#t` if Boolean |
| `(fn? x)` | `#t` if Closure or Primitive |
| `(future? x)` | `#t` if Future |
| `(zero? n)` | `#t` if 0 |
| `(even? n)` | `#t` if even |
| `(odd? n)` | `#t` if odd |
| `(positive? n)` | `#t` if > 0 |
| `(negative? n)` | `#t` if < 0 |
| `(not x)` | Boolean negation |
| `(equal? a b)` | Structural equality |
| `(eq? a b)` | Alias for `equal?` |

### String Operations
| Function | Description |
|----------|-------------|
| `(string-length s)` | Character count |
| `(string-ref s n)` | Nth character (as string) |
| `(substring s start end)` | Slice |
| `(string-append s ...)` | Concatenate |
| `(string=? a b)` | Case-sensitive equality |
| `(string-ci=? a b)` | Case-insensitive equality |
| `(string<? a b)` | Less-than |
| `(string>? a b)` | Greater-than |
| `(string-downcase s)` | Lowercase |
| `(string-upcase s)` | Uppercase |
| `(string-trim s)` | Strip leading/trailing whitespace |
| `(string-split s)` | Split on whitespace |
| `(string-split s sep)` | Split on separator |
| `(string-join parts sep)` | Join with separator |
| `(number->string n)` | Number to string |
| `(string->number s)` | String to number (or `#f`) |
| `(symbol->string s)` | Symbol name to string |
| `(string->symbol s)` | String to symbol |

### Vectors
| Function | Description |
|----------|-------------|
| `(vector x ...)` | Create vector |
| `(make-vector n)` | Vector of length n filled with `()` |
| `(make-vector n fill)` | Vector of length n filled with `fill` |
| `(vector-ref v i)` | Index (0-based) |
| `(vector-set! v i x)` | Mutate element |
| `(vector-length v)` | Length |
| `(vector? x)` | `#t` if Vector |
| `(vector->list v)` | Convert to list |
| `(list->vector lst)` | Convert to vector |
| `(vector-fill! v x)` | Fill all elements |
| `(vector-map fn v)` | Map over elements |

### I/O
| Function | Description |
|----------|-------------|
| `(display x)` | Print without newline (no quotes) |
| `(print x ...)` | Print with spaces, no newline |
| `(println x ...)` | Print with spaces and newline |
| `(newline)` | Print newline |
| `(write x)` | Print with quoting |
| `(read-line)` | Read a line from stdin (returns string, or `()` on EOF) |
| `(file->string path)` | Read file to string |
| `(string->file path content)` | Write string to file |
| `(file-exists? path)` | `#t` if file exists |
| `(delete-file path)` | Remove file |

### System
| Function | Description |
|----------|-------------|
| `(system cmd)` | Run command, print output, return exit code |
| `(shell->string cmd)` | Run command, capture stdout as string |
| `(sleep n)` | Sleep n seconds (integer or float) |
| `(usleep n)` | Sleep n milliseconds |
| `(exit)` | Exit with code 0 |
| `(exit n)` | Exit with code n |

## Examples

See `examples/` directory:
- `hello.ll` — Hello world, variables, math, lists
- `fib.ll` — Recursive Fibonacci
- `fizzbuzz.ll` — FizzBuzz with `for` and `cond`
- `php-interop.ll` — Standard library demo (replaces old PHP interop)
- `list-ports.ll` — System command output with `shell->string`
- `async.ll` — `future`, `await`, `co` async programming
