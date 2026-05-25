package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func toFloat(v Value) (float64, bool) {
	switch val := v.(type) {
	case Integer:
		return float64(val), true
	case Float:
		return float64(val), true
	default:
		return 0, false
	}
}

func isInt(v Value) bool {
	_, ok := v.(Integer)
	return ok
}

func arithBinary(fn func(a, b float64) float64, intFn func(a, b int64) int64, args []Value) (Value, error) {
	if len(args) < 1 {
		return Integer(0), nil
	}

	allInt := true
	for _, a := range args {
		if !isInt(a) {
			allInt = false
			break
		}
	}

	if allInt {
		var result int64
		switch args[0].(type) {
		case Integer:
			result = int64(args[0].(Integer))
		}

		for _, arg := range args[1:] {
			result = intFn(result, int64(arg.(Integer)))
		}
		return Integer(result), nil
	}

	var result float64
	if i, ok := toFloat(args[0]); ok {
		result = i
	}

	for _, arg := range args[1:] {
		if f, ok := toFloat(arg); ok {
			result = fn(result, f)
		}
	}

	return Float(result), nil
}

func (e *Eval) builtinAdd(args []Value) (Value, error) {
	return arithBinary(
		func(a, b float64) float64 { return a + b },
		func(a, b int64) int64 { return a + b },
		args,
	)
}

func (e *Eval) builtinSub(args []Value) (Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("- requires at least 1 argument")
	}
	return arithBinary(
		func(a, b float64) float64 { return a - b },
		func(a, b int64) int64 { return a - b },
		args,
	)
}

func (e *Eval) builtinMul(args []Value) (Value, error) {
	return arithBinary(
		func(a, b float64) float64 { return a * b },
		func(a, b int64) int64 { return a * b },
		args,
	)
}

func (e *Eval) builtinDiv(args []Value) (Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("/ requires at least 1 argument")
	}
	allInt := true
	for _, a := range args {
		if !isInt(a) {
			allInt = false
			break
		}
	}
	if allInt {
		r := int64(args[0].(Integer))
		for _, arg := range args[1:] {
			v := int64(arg.(Integer))
			if v == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			r /= v
		}
		return Integer(r), nil
	}
	r, _ := toFloat(args[0])
	for _, arg := range args[1:] {
		f, ok := toFloat(arg)
		if !ok {
			return nil, fmt.Errorf("/: non-numeric argument")
		}
		if f == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		r /= f
	}
	return Float(r), nil
}

func (e *Eval) builtinMod(args []Value) (Value, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	if len(args) != 2 {
		return nil, fmt.Errorf("%% requires 1 or 2 arguments")
	}
	a, okA := args[0].(Integer)
	b, okB := args[1].(Integer)
	if !okA || !okB {
		return nil, fmt.Errorf("%% requires integer arguments")
	}
	if b == 0 {
		return nil, fmt.Errorf("%%: division by zero")
	}
	return Integer(int64(a) % int64(b)), nil
}

func (e *Eval) builtinAbs(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("abs requires 1 argument")
	}
	switch v := args[0].(type) {
	case Integer:
		if v < 0 {
			return Integer(-int64(v)), nil
		}
		return v, nil
	case Float:
		return Float(math.Abs(float64(v))), nil
	default:
		return nil, fmt.Errorf("abs: numeric argument required")
	}
}

func (e *Eval) builtinMin(args []Value) (Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("min requires at least 1 argument")
	}
	allInt := true
	for _, a := range args {
		if !isInt(a) {
			allInt = false
			break
		}
	}
	if allInt {
		r := int64(args[0].(Integer))
		for _, a := range args[1:] {
			v := int64(a.(Integer))
			if v < r {
				r = v
			}
		}
		return Integer(r), nil
	}
	r, _ := toFloat(args[0])
	for _, a := range args[1:] {
		f, _ := toFloat(a)
		if f < r {
			r = f
		}
	}
	return Float(r), nil
}

func (e *Eval) builtinMax(args []Value) (Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("max requires at least 1 argument")
	}
	allInt := true
	for _, a := range args {
		if !isInt(a) {
			allInt = false
			break
		}
	}
	if allInt {
		r := int64(args[0].(Integer))
		for _, a := range args[1:] {
			v := int64(a.(Integer))
			if v > r {
				r = v
			}
		}
		return Integer(r), nil
	}
	r, _ := toFloat(args[0])
	for _, a := range args[1:] {
		f, _ := toFloat(a)
		if f > r {
			r = f
		}
	}
	return Float(r), nil
}

func (e *Eval) builtinExpt(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("expt requires 2 arguments")
	}
	base, okBase := args[0].(Integer)
	exp, okExp := args[1].(Integer)
	if okBase && okExp && exp >= 0 {
		r := int64(1)
		b := int64(base)
		e := int64(exp)
		for e > 0 {
			if e&1 == 1 {
				r *= b
			}
			b *= b
			e >>= 1
		}
		return Integer(r), nil
	}
	bf, _ := toFloat(args[0])
	ef, _ := toFloat(args[1])
	return Float(math.Pow(bf, ef)), nil
}

func (e *Eval) builtinSqrt(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sqrt requires 1 argument")
	}
	f, ok := toFloat(args[0])
	if !ok {
		return nil, fmt.Errorf("sqrt: numeric argument required")
	}
	return Float(math.Sqrt(f)), nil
}

func (e *Eval) builtinQuotient(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("quotient requires 2 arguments")
	}
	a, okA := args[0].(Integer)
	b, okB := args[1].(Integer)
	if !okA || !okB {
		return nil, fmt.Errorf("quotient requires integer arguments")
	}
	if b == 0 {
		return nil, fmt.Errorf("quotient: division by zero")
	}
	return Integer(int64(a) / int64(b)), nil
}

func (e *Eval) builtinRemainder(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("remainder requires 2 arguments")
	}
	a, okA := args[0].(Integer)
	b, okB := args[1].(Integer)
	if !okA || !okB {
		return nil, fmt.Errorf("remainder requires integer arguments")
	}
	if b == 0 {
		return nil, fmt.Errorf("remainder: division by zero")
	}
	return Integer(int64(a) % int64(b)), nil
}

func (e *Eval) builtinFloor(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("floor requires 1 argument")
	}
	switch v := args[0].(type) {
	case Integer:
		return v, nil
	case Float:
		return Float(math.Floor(float64(v))), nil
	default:
		return nil, fmt.Errorf("floor: numeric argument required")
	}
}

func (e *Eval) builtinCeil(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ceil requires 1 argument")
	}
	switch v := args[0].(type) {
	case Integer:
		return v, nil
	case Float:
		return Float(math.Ceil(float64(v))), nil
	default:
		return nil, fmt.Errorf("ceil: numeric argument required")
	}
}

func (e *Eval) builtinRound(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("round requires 1 argument")
	}
	switch v := args[0].(type) {
	case Integer:
		return v, nil
	case Float:
		return Float(math.Round(float64(v))), nil
	default:
		return nil, fmt.Errorf("round: numeric argument required")
	}
}

func (e *Eval) builtinNumEq(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("= requires 2 arguments")
	}
	af, okA := toFloat(args[0])
	bf, okB := toFloat(args[1])
	if !okA || !okB {
		return nil, fmt.Errorf("= requires numeric arguments")
	}
	return Boolean(af == bf), nil
}

func (e *Eval) builtinGt(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("> requires 2 arguments")
	}
	af, okA := toFloat(args[0])
	bf, okB := toFloat(args[1])
	if !okA || !okB {
		return nil, fmt.Errorf("> requires numeric arguments")
	}
	return Boolean(af > bf), nil
}

func (e *Eval) builtinLt(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("< requires 2 arguments")
	}
	af, okA := toFloat(args[0])
	bf, okB := toFloat(args[1])
	if !okA || !okB {
		return nil, fmt.Errorf("< requires numeric arguments")
	}
	return Boolean(af < bf), nil
}

func (e *Eval) builtinGte(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(">= requires 2 arguments")
	}
	af, okA := toFloat(args[0])
	bf, okB := toFloat(args[1])
	if !okA || !okB {
		return nil, fmt.Errorf(">= requires numeric arguments")
	}
	return Boolean(af >= bf), nil
}

func (e *Eval) builtinLte(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("<= requires 2 arguments")
	}
	af, okA := toFloat(args[0])
	bf, okB := toFloat(args[1])
	if !okA || !okB {
		return nil, fmt.Errorf("<= requires numeric arguments")
	}
	return Boolean(af <= bf), nil
}

// --- List operations ---

func (e *Eval) builtinCar(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("car requires 1 argument")
	}
	cons, ok := args[0].(*Cons)
	if !ok {
		return Nil, nil
	}
	return cons.Car, nil
}

func (e *Eval) builtinCdr(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("cdr requires 1 argument")
	}
	cons, ok := args[0].(*Cons)
	if !ok {
		return Nil, nil
	}
	return cons.Cdr, nil
}

func (e *Eval) builtinCons(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("cons requires 2 arguments")
	}
	return &Cons{Car: args[0], Cdr: args[1]}, nil
}

func (e *Eval) builtinList(args []Value) (Value, error) {
	return SliceToList(args), nil
}

func (e *Eval) builtinIsNull(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("null? requires 1 argument")
	}
	_, ok := args[0].(*NilType)
	return Boolean(ok), nil
}

func (e *Eval) builtinIsPair(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("pair? requires 1 argument")
	}
	_, ok := args[0].(*Cons)
	return Boolean(ok), nil
}

func (e *Eval) builtinLength(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("length requires 1 argument")
	}
	n := 0
	v := args[0]
	for {
		cons, ok := v.(*Cons)
		if !ok {
			break
		}
		n++
		v = cons.Cdr
	}
	return Integer(n), nil
}

func (e *Eval) builtinAppend(args []Value) (Value, error) {
	if len(args) == 0 {
		return Nil, nil
	}
	if len(args) == 1 {
		return args[0], nil
	}
	result := args[0]
	for _, arg := range args[1:] {
		result = appendLists(result, arg)
	}
	return result, nil
}

func appendLists(a, b Value) Value {
	if a == Nil {
		return b
	}
	cons := a.(*Cons)
	return &Cons{Car: cons.Car, Cdr: appendLists(cons.Cdr, b)}
}

func (e *Eval) builtinReverse(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("reverse requires 1 argument")
	}
	sl, ok := ListToSlice(args[0])
	if !ok {
		return nil, fmt.Errorf("reverse: argument must be a proper list")
	}
	for i, j := 0, len(sl)-1; i < j; i, j = i+1, j-1 {
		sl[i], sl[j] = sl[j], sl[i]
	}
	return SliceToList(sl), nil
}

func (e *Eval) builtinListRef(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("list-ref requires 2 arguments")
	}
	idx, ok := args[1].(Integer)
	if !ok {
		return nil, fmt.Errorf("list-ref: index must be an integer")
	}
	v := args[0]
	for i := int64(0); i < int64(idx); i++ {
		cons, ok := v.(*Cons)
		if !ok {
			return nil, fmt.Errorf("list-ref: index out of range")
		}
		v = cons.Cdr
	}
	cons, ok := v.(*Cons)
	if !ok {
		return nil, fmt.Errorf("list-ref: index out of range")
	}
	return cons.Car, nil
}

func (e *Eval) builtinListTail(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("list-tail requires 2 arguments")
	}
	idx, ok := args[1].(Integer)
	if !ok {
		return nil, fmt.Errorf("list-tail: index must be an integer")
	}
	v := args[0]
	for i := int64(0); i < int64(idx); i++ {
		cons, ok := v.(*Cons)
		if !ok {
			return nil, fmt.Errorf("list-tail: index out of range")
		}
		v = cons.Cdr
	}
	return v, nil
}

func (e *Eval) builtinTake(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("take requires 2 arguments")
	}
	n, ok := args[1].(Integer)
	if !ok {
		return nil, fmt.Errorf("take: count must be an integer")
	}
	var result []Value
	v := args[0]
	for i := int64(0); i < int64(n); i++ {
		cons, ok := v.(*Cons)
		if !ok {
			break
		}
		result = append(result, cons.Car)
		v = cons.Cdr
	}
	return SliceToList(result), nil
}

func (e *Eval) builtinDrop(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("drop requires 2 arguments")
	}
	n, ok := args[1].(Integer)
	if !ok {
		return nil, fmt.Errorf("drop: count must be an integer")
	}
	v := args[0]
	for i := int64(0); i < int64(n); i++ {
		cons, ok := v.(*Cons)
		if !ok {
			return Nil, nil
		}
		v = cons.Cdr
	}
	return v, nil
}

func (e *Eval) builtinRange(args []Value) (Value, error) {
	var start, end Integer
	if len(args) == 1 {
		start = 0
		e, ok := args[0].(Integer)
		if !ok {
			return nil, fmt.Errorf("range: argument must be an integer")
		}
		end = e
	} else if len(args) == 2 {
		s, ok := args[0].(Integer)
		if !ok {
			return nil, fmt.Errorf("range: start must be an integer")
		}
		e, ok2 := args[1].(Integer)
		if !ok2 {
			return nil, fmt.Errorf("range: end must be an integer")
		}
		start = s
		end = e
	} else {
		return nil, fmt.Errorf("range requires 1 or 2 arguments")
	}
	var result []Value
	for i := int64(start); i < int64(end); i++ {
		result = append(result, Integer(i))
	}
	return SliceToList(result), nil
}

func (e *Eval) builtinMember(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("member requires 2 arguments")
	}
	item := args[0]
	lst := args[1]
	for lst != Nil {
		cons, ok := lst.(*Cons)
		if !ok {
			return nil, fmt.Errorf("member: second argument must be a list")
		}
		if equalValue(item, cons.Car) {
			return lst, nil
		}
		lst = cons.Cdr
	}
	return Boolean(false), nil
}

func (e *Eval) builtinAssoc(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("assoc requires 2 arguments")
	}
	key := args[0]
	lst := args[1]
	for lst != Nil {
		cons, ok := lst.(*Cons)
		if !ok {
			return nil, fmt.Errorf("assoc: second argument must be a list")
		}
		pair, ok := cons.Car.(*Cons)
		if !ok {
			return nil, fmt.Errorf("assoc: list elements must be pairs")
		}
		if equalValue(key, pair.Car) {
			return pair, nil
		}
		lst = cons.Cdr
	}
	return Boolean(false), nil
}

func (e *Eval) builtinMap(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("map requires 2 arguments")
	}
	fn := args[0]
	lst := args[1]
	var result []Value
	for lst != Nil {
		cons, ok := lst.(*Cons)
		if !ok {
			return nil, fmt.Errorf("map: second argument must be a list")
		}
		val, err := e.Apply(fn, []Value{cons.Car})
		if err != nil {
			return nil, err
		}
		result = append(result, val)
		lst = cons.Cdr
	}
	return SliceToList(result), nil
}

func (e *Eval) builtinFilter(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("filter requires 2 arguments")
	}
	pred := args[0]
	lst := args[1]
	var result []Value
	for lst != Nil {
		cons, ok := lst.(*Cons)
		if !ok {
			return nil, fmt.Errorf("filter: second argument must be a list")
		}
		val, err := e.Apply(pred, []Value{cons.Car})
		if err != nil {
			return nil, err
		}
		if IsTruthy(val) {
			result = append(result, cons.Car)
		}
		lst = cons.Cdr
	}
	return SliceToList(result), nil
}

func (e *Eval) builtinFoldl(args []Value) (Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("foldl requires 3 arguments")
	}
	fn := args[0]
	acc := args[1]
	lst := args[2]
	for lst != Nil {
		cons, ok := lst.(*Cons)
		if !ok {
			return nil, fmt.Errorf("foldl: third argument must be a list")
		}
		var err error
		acc, err = e.Apply(fn, []Value{acc, cons.Car})
		if err != nil {
			return nil, err
		}
		lst = cons.Cdr
	}
	return acc, nil
}

func (e *Eval) builtinFoldr(args []Value) (Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("foldr requires 3 arguments")
	}
	fn := args[0]
	acc := args[1]
	lst := args[2]
	sl, ok := ListToSlice(lst)
	if !ok {
		return nil, fmt.Errorf("foldr: third argument must be a proper list")
	}
	for i := len(sl) - 1; i >= 0; i-- {
		var err error
		acc, err = e.Apply(fn, []Value{sl[i], acc})
		if err != nil {
			return nil, err
		}
	}
	return acc, nil
}

// --- Predicates ---

func (e *Eval) builtinIsSymbol(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("symbol? requires 1 argument")
	}
	_, ok := args[0].(*Sym)
	return Boolean(ok), nil
}

func (e *Eval) builtinIsNumber(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("number? requires 1 argument")
	}
	_, ok1 := args[0].(Integer)
	_, ok2 := args[0].(Float)
	return Boolean(ok1 || ok2), nil
}

func (e *Eval) builtinIsInteger(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("integer? requires 1 argument")
	}
	_, ok := args[0].(Integer)
	return Boolean(ok), nil
}

func (e *Eval) builtinIsFloat(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("float? requires 1 argument")
	}
	_, ok := args[0].(Float)
	return Boolean(ok), nil
}

func (e *Eval) builtinIsString(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string? requires 1 argument")
	}
	_, ok := args[0].(String)
	return Boolean(ok), nil
}

func (e *Eval) builtinIsBoolean(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("boolean? requires 1 argument")
	}
	_, ok := args[0].(Boolean)
	return Boolean(ok), nil
}

func (e *Eval) builtinIsList(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("list? requires 1 argument")
	}
	return Boolean(isProperList(args[0])), nil
}

func isProperList(v Value) bool {
	for v != Nil {
		_, ok := v.(*Cons)
		if !ok {
			return false
		}
		v = v.(*Cons).Cdr
	}
	return true
}

func (e *Eval) builtinIsFn(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("fn? requires 1 argument")
	}
	_, ok1 := args[0].(*Primitive)
	_, ok2 := args[0].(*Closure)
	_, ok3 := args[0].(*Macro)
	return Boolean(ok1 || ok2 || ok3), nil
}

func (e *Eval) builtinIsFuture(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("future? requires 1 argument")
	}
	_, ok := args[0].(*Future)
	return Boolean(ok), nil
}

func (e *Eval) builtinNot(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("not requires 1 argument")
	}
	return Boolean(!IsTruthy(args[0])), nil
}

func (e *Eval) builtinZero(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("zero? requires 1 argument")
	}
	switch v := args[0].(type) {
	case Integer:
		return Boolean(v == 0), nil
	case Float:
		return Boolean(v == 0), nil
	default:
		return nil, fmt.Errorf("zero? requires numeric argument")
	}
}

func (e *Eval) builtinEven(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("even? requires 1 argument")
	}
	v, ok := args[0].(Integer)
	if !ok {
		return nil, fmt.Errorf("even? requires integer argument")
	}
	return Boolean(int64(v)%2 == 0), nil
}

func (e *Eval) builtinOdd(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("odd? requires 1 argument")
	}
	v, ok := args[0].(Integer)
	if !ok {
		return nil, fmt.Errorf("odd? requires integer argument")
	}
	return Boolean(int64(v)%2 == 1), nil
}

func (e *Eval) builtinPositive(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("positive? requires 1 argument")
	}
	f, ok := toFloat(args[0])
	if !ok {
		return nil, fmt.Errorf("positive? requires numeric argument")
	}
	return Boolean(f > 0), nil
}

func (e *Eval) builtinNegative(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("negative? requires 1 argument")
	}
	f, ok := toFloat(args[0])
	if !ok {
		return nil, fmt.Errorf("negative? requires numeric argument")
	}
	return Boolean(f < 0), nil
}

func (e *Eval) builtinInc(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("inc requires 1 argument")
	}
	switch v := args[0].(type) {
	case Integer:
		return Integer(int64(v) + 1), nil
	case Float:
		return Float(float64(v) + 1), nil
	default:
		return nil, fmt.Errorf("inc requires numeric argument")
	}
}

func (e *Eval) builtinDec(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("dec requires 1 argument")
	}
	switch v := args[0].(type) {
	case Integer:
		return Integer(int64(v) - 1), nil
	case Float:
		return Float(float64(v) - 1), nil
	default:
		return nil, fmt.Errorf("dec requires numeric argument")
	}
}

func (e *Eval) builtinEqual(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("equal? requires 2 arguments")
	}
	return Boolean(equalValue(args[0], args[1])), nil
}

func (e *Eval) builtinEq(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("eq? requires 2 arguments")
	}
	return Boolean(equalValue(args[0], args[1])), nil
}

func equalValue(a, b Value) bool {
	switch av := a.(type) {
	case *NilType:
		_, ok := b.(*NilType)
		return ok
	case Integer:
		bv, ok := b.(Integer)
		return ok && av == bv
	case Float:
		bv, ok := b.(Float)
		return ok && av == bv
	case String:
		bv, ok := b.(String)
		return ok && av == bv
	case Boolean:
		bv, ok := b.(Boolean)
		return ok && av == bv
	case *Sym:
		bv, ok := b.(*Sym)
		return ok && av.Name == bv.Name
	case *Cons:
		bv, ok := b.(*Cons)
		if !ok {
			return false
		}
		return equalValue(av.Car, bv.Car) && equalValue(av.Cdr, bv.Cdr)
	case *Vector:
		bv, ok := b.(*Vector)
		if !ok || len(av.Items) != len(bv.Items) {
			return false
		}
		for i := range av.Items {
			if !equalValue(av.Items[i], bv.Items[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// --- String operations ---

func (e *Eval) builtinStringLength(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string-length requires 1 argument")
	}
	s, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("string-length requires string argument")
	}
	return Integer(len([]rune(string(s)))), nil
}

func (e *Eval) builtinStringRef(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("string-ref requires 2 arguments")
	}
	s, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("string-ref requires string argument")
	}
	idx, ok2 := args[1].(Integer)
	if !ok2 {
		return nil, fmt.Errorf("string-ref requires integer index")
	}
	runes := []rune(string(s))
	i := int64(idx)
	if i < 0 || i >= int64(len(runes)) {
		return nil, fmt.Errorf("string-ref: index out of range")
	}
	return String(string(runes[i])), nil
}

func (e *Eval) builtinSubstring(args []Value) (Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("substring requires 2 or 3 arguments")
	}
	s, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("substring requires string argument")
	}
	start, ok2 := args[1].(Integer)
	if !ok2 {
		return nil, fmt.Errorf("substring: start must be an integer")
	}
	runes := []rune(string(s))
	si := int64(start)
	if si < 0 || si > int64(len(runes)) {
		return nil, fmt.Errorf("substring: start index out of range")
	}
	if len(args) == 2 {
		return String(string(runes[si:])), nil
	}
	end, ok3 := args[2].(Integer)
	if !ok3 {
		return nil, fmt.Errorf("substring: end must be an integer")
	}
	ei := int64(end)
	if ei < si || ei > int64(len(runes)) {
		return nil, fmt.Errorf("substring: end index out of range")
	}
	return String(string(runes[si:ei])), nil
}

func (e *Eval) builtinStringAppend(args []Value) (Value, error) {
	var b strings.Builder
	for _, arg := range args {
		s, ok := arg.(String)
		if !ok {
			return nil, fmt.Errorf("string-append requires string arguments")
		}
		b.WriteString(string(s))
	}
	return String(b.String()), nil
}

func (e *Eval) builtinStringEq(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("string=? requires 2 arguments")
	}
	a, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("string=? requires string arguments")
	}
	b, ok2 := args[1].(String)
	if !ok2 {
		return nil, fmt.Errorf("string=? requires string arguments")
	}
	return Boolean(a == b), nil
}

func (e *Eval) builtinStringCiEq(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("string-ci=? requires 2 arguments")
	}
	a, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("string-ci=? requires string arguments")
	}
	b, ok2 := args[1].(String)
	if !ok2 {
		return nil, fmt.Errorf("string-ci=? requires string arguments")
	}
	return Boolean(strings.EqualFold(string(a), string(b))), nil
}

func (e *Eval) builtinStringLt(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("string<? requires 2 arguments")
	}
	a, ok := args[0].(String)
	b, ok2 := args[1].(String)
	if !ok || !ok2 {
		return nil, fmt.Errorf("string<? requires string arguments")
	}
	return Boolean(string(a) < string(b)), nil
}

func (e *Eval) builtinStringGt(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("string>? requires 2 arguments")
	}
	a, ok := args[0].(String)
	b, ok2 := args[1].(String)
	if !ok || !ok2 {
		return nil, fmt.Errorf("string>? requires string arguments")
	}
	return Boolean(string(a) > string(b)), nil
}

func (e *Eval) builtinStringDowncase(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string-downcase requires 1 argument")
	}
	s, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("string-downcase requires string argument")
	}
	return String(strings.ToLower(string(s))), nil
}

func (e *Eval) builtinStringUpcase(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string-upcase requires 1 argument")
	}
	s, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("string-upcase requires string argument")
	}
	return String(strings.ToUpper(string(s))), nil
}

func (e *Eval) builtinStringTrim(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string-trim requires 1 argument")
	}
	s, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("string-trim requires string argument")
	}
	return String(strings.TrimSpace(string(s))), nil
}

func (e *Eval) builtinStringSplit(args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("string-split requires 1 or 2 arguments")
	}
	s, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("string-split requires string argument")
	}
	if len(args) == 1 {
		fields := strings.Fields(string(s))
		var result []Value
		for _, f := range fields {
			result = append(result, String(f))
		}
		return SliceToList(result), nil
	}
	delim, ok2 := args[1].(String)
	if !ok2 {
		return nil, fmt.Errorf("string-split: delimiter must be a string")
	}
	parts := strings.Split(string(s), string(delim))
	var result []Value
	for _, p := range parts {
		result = append(result, String(p))
	}
	return SliceToList(result), nil
}

func (e *Eval) builtinStringJoin(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("string-join requires 2 arguments")
	}
	separator, ok := args[1].(String)
	if !ok {
		return nil, fmt.Errorf("string-join: separator must be a string")
	}
	sl, ok := ListToSlice(args[0])
	if !ok {
		return nil, fmt.Errorf("string-join: first argument must be a list of strings")
	}
	var parts []string
	for _, v := range sl {
		s, ok := v.(String)
		if !ok {
			return nil, fmt.Errorf("string-join: list must contain strings")
		}
		parts = append(parts, string(s))
	}
	return String(strings.Join(parts, string(separator))), nil
}

func (e *Eval) builtinNumberToString(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("number->string requires 1 argument")
	}
	switch v := args[0].(type) {
	case Integer:
		return String(fmt.Sprintf("%d", int64(v))), nil
	case Float:
		return String(fmt.Sprintf("%g", float64(v))), nil
	default:
		return nil, fmt.Errorf("number->string requires numeric argument")
	}
}

func (e *Eval) builtinStringToNumber(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string->number requires 1 argument")
	}
	s, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("string->number requires string argument")
	}
	str := string(s)
	if strings.Contains(str, ".") {
		f, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return Boolean(false), nil
		}
		return Float(f), nil
	}
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return Boolean(false), nil
	}
	return Integer(n), nil
}

func (e *Eval) builtinSymbolToString(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("symbol->string requires 1 argument")
	}
	sym, ok := args[0].(*Sym)
	if !ok {
		return nil, fmt.Errorf("symbol->string requires symbol argument")
	}
	return String(sym.Name), nil
}

func (e *Eval) builtinStringToSymbol(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string->symbol requires 1 argument")
	}
	s, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("string->symbol requires string argument")
	}
	return &Sym{Name: string(s)}, nil
}

// --- I/O ---

func (e *Eval) builtinDisplay(args []Value) (Value, error) {
	for _, a := range args {
		Display(e.w, a)
	}
	return Nil, nil
}

func (e *Eval) builtinWrite(args []Value) (Value, error) {
	for _, a := range args {
		fmt.Fprint(e.w, Sprint(a))
	}
	return Nil, nil
}

func (e *Eval) builtinPrintln(args []Value) (Value, error) {
	for i, a := range args {
		if i > 0 {
			fmt.Fprint(e.w, " ")
		}
		Display(e.w, a)
	}
	fmt.Fprintln(e.w)
	return Nil, nil
}

func (e *Eval) builtinPrint(args []Value) (Value, error) {
	for i, a := range args {
		if i > 0 {
			fmt.Fprint(e.w, " ")
		}
		Display(e.w, a)
	}
	return Nil, nil
}

func (e *Eval) builtinNewline(args []Value) (Value, error) {
	fmt.Println()
	return Nil, nil
}

func (e *Eval) builtinReadLine(args []Value) (Value, error) {
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return Nil, nil
	}
	return String(strings.TrimRight(line, "\n\r")), nil
}

// --- File I/O ---

func (e *Eval) builtinFileToString(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("file->string requires 1 argument")
	}
	name, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("file->string requires string argument")
	}
	data, err := os.ReadFile(string(name))
	if err != nil {
		return nil, err
	}
	return String(string(data)), nil
}

func (e *Eval) builtinStringToFile(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("string->file requires 2 arguments")
	}
	name, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("string->file: filename must be a string")
	}
	content, ok2 := args[1].(String)
	if !ok2 {
		return nil, fmt.Errorf("string->file: content must be a string")
	}
	err := os.WriteFile(string(name), []byte(string(content)), 0644)
	if err != nil {
		return nil, err
	}
	return Nil, nil
}

func (e *Eval) builtinFileExists(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("file-exists? requires 1 argument")
	}
	name, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("file-exists? requires string argument")
	}
	_, err := os.Stat(string(name))
	return Boolean(err == nil), nil
}

func (e *Eval) builtinDeleteFile(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("delete-file requires 1 argument")
	}
	name, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("delete-file requires string argument")
	}
	err := os.Remove(string(name))
	if err != nil {
		return nil, err
	}
	return Nil, nil
}

func (e *Eval) builtinSleep(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sleep requires 1 argument")
	}
	switch v := args[0].(type) {
	case Integer:
		time.Sleep(time.Duration(v) * time.Second)
	case Float:
		time.Sleep(time.Duration(v * 1e9))
	default:
		return nil, fmt.Errorf("sleep: argument must be a number")
	}
	return Nil, nil
}

func (e *Eval) builtinUsleep(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("usleep requires 1 argument")
	}
	v, ok := args[0].(Integer)
	if !ok {
		return nil, fmt.Errorf("usleep: argument must be an integer")
	}
	time.Sleep(time.Duration(v) * time.Millisecond)
	return Nil, nil
}

func (e *Eval) builtinExit(args []Value) (Value, error) {
	code := 0
	if len(args) > 0 {
		if c, ok := args[0].(Integer); ok {
			code = int(c)
		}
	}
	os.Exit(code)
	return Nil, nil
}

func (e *Eval) builtinGetFileDir(args []Value) (Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("get-file-dir requires 0 arguments")
	}
	if e.currentFile == "" {
		return String(""), nil
	}
	abs, err := filepath.Abs(e.currentFile)
	if err != nil {
		return String(""), nil
	}
	return String(filepath.Dir(abs)), nil
}

// --- Vector operations ---

func (e *Eval) builtinVector(args []Value) (Value, error) {
	items := make([]Value, len(args))
	copy(items, args)
	return &Vector{Items: items}, nil
}

func (e *Eval) builtinMakeVector(args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("make-vector requires 1 or 2 arguments")
	}
	n, ok := args[0].(Integer)
	if !ok {
		return nil, fmt.Errorf("make-vector: size must be an integer")
	}
	var fill Value = Nil
	if len(args) == 2 {
		fill = args[1]
	}
	size := int64(n)
	if size < 0 {
		return nil, fmt.Errorf("make-vector: size must be non-negative")
	}
	items := make([]Value, size)
	for i := int64(0); i < size; i++ {
		items[i] = fill
	}
	return &Vector{Items: items}, nil
}

func (e *Eval) builtinVectorRef(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("vector-ref requires 2 arguments")
	}
	v, ok := args[0].(*Vector)
	if !ok {
		return nil, fmt.Errorf("vector-ref: first argument must be a vector")
	}
	i, ok := args[1].(Integer)
	if !ok {
		return nil, fmt.Errorf("vector-ref: index must be an integer")
	}
	idx := int64(i)
	if idx < 0 || idx >= int64(len(v.Items)) {
		return nil, fmt.Errorf("vector-ref: index out of range")
	}
	return v.Items[idx], nil
}

func (e *Eval) builtinVectorSet(args []Value) (Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("vector-set! requires 3 arguments")
	}
	v, ok := args[0].(*Vector)
	if !ok {
		return nil, fmt.Errorf("vector-set!: first argument must be a vector")
	}
	i, ok := args[1].(Integer)
	if !ok {
		return nil, fmt.Errorf("vector-set!: index must be an integer")
	}
	idx := int64(i)
	if idx < 0 || idx >= int64(len(v.Items)) {
		return nil, fmt.Errorf("vector-set!: index out of range")
	}
	v.Items[idx] = args[2]
	return args[2], nil
}

func (e *Eval) builtinVectorLength(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("vector-length requires 1 argument")
	}
	v, ok := args[0].(*Vector)
	if !ok {
		return nil, fmt.Errorf("vector-length: argument must be a vector")
	}
	return Integer(len(v.Items)), nil
}

func (e *Eval) builtinIsVector(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("vector? requires 1 argument")
	}
	_, ok := args[0].(*Vector)
	return Boolean(ok), nil
}

func (e *Eval) builtinVectorToList(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("vector->list requires 1 argument")
	}
	v, ok := args[0].(*Vector)
	if !ok {
		return nil, fmt.Errorf("vector->list: argument must be a vector")
	}
	return SliceToList(v.Items), nil
}

func (e *Eval) builtinListToVector(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("list->vector requires 1 argument")
	}
	sl, ok := ListToSlice(args[0])
	if !ok {
		return nil, fmt.Errorf("list->vector: argument must be a proper list")
	}
	return &Vector{Items: sl}, nil
}

func (e *Eval) builtinVectorFill(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("vector-fill! requires 2 arguments")
	}
	v, ok := args[0].(*Vector)
	if !ok {
		return nil, fmt.Errorf("vector-fill!: first argument must be a vector")
	}
	for i := range v.Items {
		v.Items[i] = args[1]
	}
	return Nil, nil
}

func (e *Eval) builtinVectorMap(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("vector-map requires 2 arguments")
	}
	fn := args[0]
	v, ok := args[1].(*Vector)
	if !ok {
		return nil, fmt.Errorf("vector-map: second argument must be a vector")
	}
	result := make([]Value, len(v.Items))
	for i, item := range v.Items {
		val, err := e.Apply(fn, []Value{item})
		if err != nil {
			return nil, err
		}
		result[i] = val
	}
	return &Vector{Items: result}, nil
}

// --- OOP ---

func (e *Eval) builtinMakeClass(args []Value) (Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("make-class requires at least 2 arguments")
	}
	nameSym, ok := args[0].(*Sym)
	if !ok {
		return nil, fmt.Errorf("make-class: first argument must be a symbol")
	}

	var parent *ClassType
	if args[1] != Nil {
		p, ok := args[1].(*ClassType)
		if !ok {
			return nil, fmt.Errorf("make-class: parent must be a class or nil")
		}
		parent = p
	}

	var slotDefs []*SlotDef
	if len(args) >= 3 && args[2] != Nil {
		slotsList, ok := args[2].(*Cons)
		if !ok {
			return nil, fmt.Errorf("make-class: slot definitions must be a list")
		}
		slotItems, ok := ListToSlice(slotsList)
		if !ok {
			return nil, fmt.Errorf("make-class: invalid slot definitions list")
		}
		for _, item := range slotItems {
			defCons, ok := item.(*Cons)
			if !ok {
				return nil, fmt.Errorf("make-class: each slot def must be a list (name . defaults)")
			}
			defSlice, ok := ListToSlice(defCons)
			if !ok {
				return nil, fmt.Errorf("make-class: invalid slot definition")
			}
			if len(defSlice) < 1 {
				return nil, fmt.Errorf("make-class: slot definition needs at least a name")
			}
			nameSym, ok := defSlice[0].(*Sym)
			if !ok {
				return nil, fmt.Errorf("make-class: slot name must be a symbol")
			}
			sd := &SlotDef{Name: nameSym.Name, Default: Nil}
			if len(defSlice) >= 2 {
				sd.Default = defSlice[1]
			}
			slotDefs = append(slotDefs, sd)
		}
	}

	// Compute full slot list: parent slots first, then own
	var allSlots []*SlotDef
	if parent != nil {
		allSlots = append(allSlots, parent.Slots...)
	}
	allSlots = append(allSlots, slotDefs...)

	class := &ClassType{
		Name:    nameSym.Name,
		Parent:  parent,
		Slots:   allSlots,
		Methods: make(map[string]*Closure),
	}
	return class, nil
}

func lookupMethod(class *ClassType, name string) *Closure {
	for c := class; c != nil; c = c.Parent {
		if m, ok := c.Methods[name]; ok {
			return m
		}
	}
	return nil
}

func (e *Eval) builtinNew(args []Value) (Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("new requires at least 1 argument (a class)")
	}
	class, ok := args[0].(*ClassType)
	if !ok {
		return nil, fmt.Errorf("new: first argument must be a class")
	}

	// Build slot values: default all, then override with keyword args
	data := make([]Value, len(class.Slots))
	for i, sd := range class.Slots {
		data[i] = sd.Default
	}

	// Process keyword args: (new Dog 'name "Rex" 'breed "Husky")
	for i := 1; i+1 < len(args); i += 2 {
		keySym, ok := args[i].(*Sym)
		if !ok {
			return nil, fmt.Errorf("new: slot keys must be symbols")
		}
		found := false
		for j, sd := range class.Slots {
			if sd.Name == keySym.Name {
				data[j] = args[i+1]
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("new: unknown slot '%s' for class %s", keySym.Name, class.Name)
		}
	}

	inst := &Instance{
		Class: class,
		Data:  data,
	}
	return inst, nil
}

func (e *Eval) builtinSend(args []Value) (Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("send requires at least 2 arguments (instance method-name)")
	}
	inst, ok := args[0].(*Instance)
	if !ok {
		return nil, fmt.Errorf("send: first argument must be an instance")
	}
	methodSym, ok := args[1].(*Sym)
	if !ok {
		return nil, fmt.Errorf("send: method name must be a symbol")
	}
	method := lookupMethod(inst.Class, methodSym.Name)
	if method == nil {
		return nil, fmt.Errorf("send: no method '%s' on %s", methodSym.Name, inst.Class.Name)
	}
	callArgs := []Value{inst}
	callArgs = append(callArgs, args[2:]...)
	return e.Apply(method, callArgs)
}

func (e *Eval) builtinSlotRef(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("slot-ref requires 2 arguments")
	}
	inst, ok := args[0].(*Instance)
	if !ok {
		return nil, fmt.Errorf("slot-ref: first argument must be an instance")
	}
	nameSym, ok := args[1].(*Sym)
	if !ok {
		return nil, fmt.Errorf("slot-ref: second argument must be a symbol")
	}
	for i, sd := range inst.Class.Slots {
		if sd.Name == nameSym.Name {
			return inst.Data[i], nil
		}
	}
	return nil, fmt.Errorf("slot-ref: no slot '%s' on %s", nameSym.Name, inst.Class.Name)
}

func (e *Eval) builtinSlotSet(args []Value) (Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("slot-set! requires 3 arguments")
	}
	inst, ok := args[0].(*Instance)
	if !ok {
		return nil, fmt.Errorf("slot-set!: first argument must be an instance")
	}
	nameSym, ok := args[1].(*Sym)
	if !ok {
		return nil, fmt.Errorf("slot-set!: second argument must be a symbol")
	}
	for i, sd := range inst.Class.Slots {
		if sd.Name == nameSym.Name {
			inst.Data[i] = args[2]
			return args[2], nil
		}
	}
	return nil, fmt.Errorf("slot-set!: no slot '%s' on %s", nameSym.Name, inst.Class.Name)
}

func (e *Eval) builtinInstanceOf(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("instance? requires 1 argument")
	}
	_, ok := args[0].(*Instance)
	return Boolean(ok), nil
}

func (e *Eval) builtinClassOf(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("class-of requires 1 argument")
	}
	inst, ok := args[0].(*Instance)
	if !ok {
		return nil, fmt.Errorf("class-of: argument must be an instance")
	}
	return inst.Class, nil
}

func (e *Eval) builtinAddMethod(args []Value) (Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("add-method requires 3 arguments (class name function)")
	}
	class, ok := args[0].(*ClassType)
	if !ok {
		return nil, fmt.Errorf("add-method: first argument must be a class")
	}
	methodSym, ok := args[1].(*Sym)
	if !ok {
		return nil, fmt.Errorf("add-method: second argument must be a symbol")
	}
	fn, ok := args[2].(*Closure)
	if !ok {
		return nil, fmt.Errorf("add-method: third argument must be a function")
	}
	class.Methods[methodSym.Name] = fn
	return fn, nil
}

// --- HTTP Server ---

func (e *Eval) builtinHttpCreateServer(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("http/create-server requires 2 arguments (host port)")
	}
	host, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("http/create-server: host must be a string")
	}
	port, ok := args[1].(Integer)
	if !ok {
		return nil, fmt.Errorf("http/create-server: port must be an integer")
	}
	return &HttpServer{
		Host:    string(host),
		Port:    int(port),
		Handler: Nil,
	}, nil
}

func (e *Eval) builtinHttpSetHandler(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("http/set-handler requires 2 arguments (server handler)")
	}
	server, ok := args[0].(*HttpServer)
	if !ok {
		return nil, fmt.Errorf("http/set-handler: first argument must be an http-server")
	}
	switch args[1].(type) {
	case *Closure, *Primitive:
		server.Handler = args[1]
	default:
		return nil, fmt.Errorf("http/set-handler: second argument must be a function")
	}
	return Nil, nil
}

func (e *Eval) builtinHttpStartServer(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("http/start-server requires 1 argument")
	}
	server, ok := args[0].(*HttpServer)
	if !ok {
		return nil, fmt.Errorf("http/start-server: argument must be an http-server")
	}
	if server.Handler == nil {
		return nil, fmt.Errorf("http/start-server: no handler set")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body.Close()

		headers := make(map[string]string)
		for key, vals := range r.Header {
			headers[key] = strings.Join(vals, ", ")
		}

		req := &HttpRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Headers: headers,
			Body:    string(bodyBytes),
		}

		result, err := e.Apply(server.Handler, []Value{req})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp, ok := result.(*HttpResponse)
		if !ok {
			http.Error(w, "handler must return an http-response", http.StatusInternalServerError)
			return
		}

		for key, val := range resp.Headers {
			w.Header().Set(key, val)
		}
		w.WriteHeader(resp.Status)
		w.Write([]byte(resp.Body))
	})

	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
	srv := &http.Server{Addr: addr, Handler: mux}

	fmt.Fprintf(e.w, "HTTP server listening on %s\n", addr)
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return Nil, err
	}
	return Nil, nil
}

func (e *Eval) builtinHttpRequestMethod(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("http/request-method requires 1 argument")
	}
	req, ok := args[0].(*HttpRequest)
	if !ok {
		return nil, fmt.Errorf("http/request-method: argument must be an http-request")
	}
	return String(req.Method), nil
}

func (e *Eval) builtinHttpRequestPath(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("http/request-path requires 1 argument")
	}
	req, ok := args[0].(*HttpRequest)
	if !ok {
		return nil, fmt.Errorf("http/request-path: argument must be an http-request")
	}
	return String(req.Path), nil
}

func (e *Eval) builtinHttpRequestHeaders(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("http/request-headers requires 1 argument")
	}
	req, ok := args[0].(*HttpRequest)
	if !ok {
		return nil, fmt.Errorf("http/request-headers: argument must be an http-request")
	}
	var result Value = Nil
	for key, val := range req.Headers {
		result = &Cons{
			Car: &Cons{Car: String(key), Cdr: String(val)},
			Cdr: result,
		}
	}
	return result, nil
}

func (e *Eval) builtinHttpRequestBody(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("http/request-body requires 1 argument")
	}
	req, ok := args[0].(*HttpRequest)
	if !ok {
		return nil, fmt.Errorf("http/request-body: argument must be an http-request")
	}
	return String(req.Body), nil
}

func (e *Eval) builtinHttpMakeResponse(args []Value) (Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("http/make-response requires 3 arguments (status headers body)")
	}
	status, ok := args[0].(Integer)
	if !ok {
		return nil, fmt.Errorf("http/make-response: status must be an integer")
	}

	headers := make(map[string]string)
	hdrs := args[1]
	for hdrs != Nil {
		pairCons, ok := hdrs.(*Cons)
		if !ok {
			break
		}
		pair, ok := pairCons.Car.(*Cons)
		if ok {
			key, okK := pair.Car.(String)
			if okK {
				switch val := pair.Cdr.(type) {
				case String:
					headers[string(key)] = string(val)
				case *Cons:
					if v, ok := val.Car.(String); ok {
						headers[string(key)] = string(v)
					}
				}
			}
		}
		hdrs = pairCons.Cdr
	}

	body, ok := args[2].(String)
	if !ok {
		return nil, fmt.Errorf("http/make-response: body must be a string")
	}

	return &HttpResponse{
		Status:  int(status),
		Headers: headers,
		Body:    string(body),
	}, nil
}

func (e *Eval) builtinHttpResponseStatus(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("http/response-status requires 1 argument")
	}
	resp, ok := args[0].(*HttpResponse)
	if !ok {
		return nil, fmt.Errorf("http/response-status: argument must be an http-response")
	}
	return Integer(resp.Status), nil
}

func (e *Eval) builtinHttpResponseHeaders(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("http/response-headers requires 1 argument")
	}
	resp, ok := args[0].(*HttpResponse)
	if !ok {
		return nil, fmt.Errorf("http/response-headers: argument must be an http-response")
	}
	var result Value = Nil
	for key, val := range resp.Headers {
		result = &Cons{
			Car: &Cons{Car: String(key), Cdr: String(val)},
			Cdr: result,
		}
	}
	return result, nil
}

func (e *Eval) builtinHttpResponseBody(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("http/response-body requires 1 argument")
	}
	resp, ok := args[0].(*HttpResponse)
	if !ok {
		return nil, fmt.Errorf("http/response-body: argument must be an http-response")
	}
	return String(resp.Body), nil
}

// --- System interface ---

func (e *Eval) builtinSystem(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("system requires 1 argument")
	}
	cmdStr, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("system: command must be a string")
	}
	cmd := exec.Command("sh", "-c", string(cmdStr))
	cmd.Stdout = e.w
	cmd.Stderr = e.w
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return Integer(exitErr.ExitCode()), nil
		}
		return Integer(-1), nil
	}
	return Integer(0), nil
}

func (e *Eval) builtinShellToString(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("shell->string requires 1 argument")
	}
	cmdStr, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("shell->string: command must be a string")
	}
	cmd := exec.Command("sh", "-c", string(cmdStr))
	out, err := cmd.Output()
	if err != nil {
		return String(""), nil
	}
	return String(string(out)), nil
}
