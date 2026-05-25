#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
LL="$ROOT/ll"
TMPDIR=$(mktemp -d /tmp/ll-test-errors.XXXXXX)
PASS=0
FAIL=0

red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }

cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

test_error() {
	local label="$1" file="$2" expected="$3"
	printf '  %-55s ... ' "$label"
	local output
	output="$("$LL" "$file" 2>&1 || true)"
	# strip ANSI color codes
	output="$(echo "$output" | sed 's/\x1b\[[0-9;]*m//g')"
	# strip "Error: " prefix
	output="$(echo "$output" | sed 's/^Error: //')"
	if echo "$output" | grep -qE "$expected"; then
		green "PASS"
		PASS=$((PASS + 1))
	else
		red "FAIL"
		echo "       expected regex: $expected" >&2
		echo "       got:            $output" >&2
		FAIL=$((FAIL + 1))
	fi
}

echo "=== Error message format tests ==="
echo ""

echo "Build"
printf '  %-55s ... ' "go build"
go build -o "$LL" "$ROOT" && green "PASS" || red "FAIL"

echo ""
echo "Compiler errors (lexer + parser)"

cat > "$TMPDIR/lexer_unterm_str.ll" << 'EOF'
(define x "hello
EOF
test_error "lexer: unterminated string" \
  "$TMPDIR/lexer_unterm_str.ll" \
  ":[0-9]+: unterminated string"

cat > "$TMPDIR/parser_extra_paren.ll" << 'EOF'
(define x 1)
)
EOF
test_error "parser: unexpected ')'" \
  "$TMPDIR/parser_extra_paren.ll" \
  ":[0-9]+: unexpected '\)'"

cat > "$TMPDIR/parser_unterm_list.ll" << 'EOF'
(define x
EOF
test_error "parser: unterminated list" \
  "$TMPDIR/parser_unterm_list.ll" \
  ":[0-9]+: unterminated list"

echo ""
echo "Special form errors"

cat > "$TMPDIR/define_no_args.ll" << 'EOF'
(define)
EOF
test_error "special: (define)" \
  "$TMPDIR/define_no_args.ll" \
  ":[0-9]+: define requires at least 2 arguments"

cat > "$TMPDIR/if_no_args.ll" << 'EOF'
(if)
EOF
test_error "special: (if)" \
  "$TMPDIR/if_no_args.ll" \
  ":[0-9]+: if requires at least 2 arguments"

cat > "$TMPDIR/lambda_no_body.ll" << 'EOF'
(lambda (x))
EOF
test_error "special: (lambda (x))" \
  "$TMPDIR/lambda_no_body.ll" \
  ":[0-9]+: lambda requires at least one body expression"

cat > "$TMPDIR/set_no_args.ll" << 'EOF'
(set! x)
EOF
test_error "special: (set! x)" \
  "$TMPDIR/set_no_args.ll" \
  ":[0-9]+: set! requires 2 arguments"

echo ""
echo "Builtin errors"

cat > "$TMPDIR/div_by_zero.ll" << 'EOF'
(/ 1 0)
EOF
test_error "builtin: division by zero" \
  "$TMPDIR/div_by_zero.ll" \
  ":[0-9]+: division by zero"

cat > "$TMPDIR/string_length_type.ll" << 'EOF'
(string-length 42)
EOF
test_error "builtin: (string-length 42)" \
  "$TMPDIR/string_length_type.ll" \
  ":[0-9]+: string-length requires string argument"

cat > "$TMPDIR/string_ref_type.ll" << 'EOF'
(string-ref "abc" "x")
EOF
test_error "builtin: (string-ref \"abc\" \"x\")" \
  "$TMPDIR/string_ref_type.ll" \
  ":[0-9]+: string-ref requires integer index"

echo ""
echo "Variable errors"

cat > "$TMPDIR/undefined_var.ll" << 'EOF'
(undefined-func 42)
EOF
test_error "variable: undefined" \
  "$TMPDIR/undefined_var.ll" \
  ":[0-9]+: undefined variable: undefined-func"

cat > "$TMPDIR/set_undef.ll" << 'EOF'
(set! nonexistent 1)
EOF
test_error "set!: undefined variable" \
  "$TMPDIR/set_undef.ll" \
  ":[0-9]+: cannot set! undefined variable: nonexistent"

echo ""
echo "Argument errors"

cat > "$TMPDIR/wrong_arg_count.ll" << 'EOF'
(define (f x) x)
(f 1 2 3)
EOF
test_error "fn: wrong arg count" \
  "$TMPDIR/wrong_arg_count.ll" \
  ":[0-9]+: expected 1 arguments, got 3"

cat > "$TMPDIR/not_callable.ll" << 'EOF'
(define x 42)
(x)
EOF
test_error "fn: not callable" \
  "$TMPDIR/not_callable.ll" \
  ":[0-9]+: not callable: 42"

echo ""
echo "Error in required file"

mkdir -p "$TMPDIR/sub"
cat > "$TMPDIR/sub/lib.ll" << 'EOF'
(lambda (x))
EOF
cat > "$TMPDIR/require_error.ll" << 'EOF'
(require "sub/lib.ll")
EOF
printf '  %-55s ... ' "require: error in required file"
output="$(cd "$TMPDIR" && "$LL" "require_error.ll" 2>&1 || true)"
output="$(echo "$output" | sed 's/\x1b\[[0-9;]*m//g')"
output="$(echo "$output" | sed 's/^Error: //')"
if echo "$output" | grep -qE "sub/lib\.ll:[0-9]+: lambda requires at least one body expression"; then
	green "PASS"
	PASS=$((PASS + 1))
else
	red "FAIL"
	echo "       expected regex: sub/lib\\.ll:[0-9]+: lambda requires at least one body expression" >&2
	echo "       got:            $output" >&2
	FAIL=$((FAIL + 1))
fi

cat > "$TMPDIR/require_not_found.ll" << 'EOF'
(require "nonexistent.ll")
EOF
test_error "require: file not found" \
  "$TMPDIR/require_not_found.ll" \
  ":[0-9]+: cannot find file: nonexistent.ll"

echo ""
echo "Error propagation through nested function calls"

cat > "$TMPDIR/nested_error.ll" << 'EOF'
(define (inner x)
  (/ 1 x))
(define (outer y)
  (inner y))
(outer 0)
EOF
test_error "nested: inner function error" \
  "$TMPDIR/nested_error.ll" \
  "nested_error.ll:2: division by zero"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] || exit 1
