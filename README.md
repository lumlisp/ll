# Lum Lisp

A Lisp dialect implemented in Go.

```
go build -o ll .
./ll              # REPL
./ll file.ll      # run file
./ll --help       # usage
```

## Quick start

```sh
go build -o ll .
./ll examples/hello.ll
```

## Documentation

See [docs/reference.md](docs/reference.md) for the language reference.

## Testing

```sh
go test ./...      # unit tests
./test-a-ll        # full test suite (unit tests + examples)
```
