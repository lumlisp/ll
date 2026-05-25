package main

import (
	"fmt"
	"strings"
)

type Value interface {
	isValue()
	String() string
}

var Nil = &NilType{}

type NilType struct{}

func (*NilType) isValue()       {}
func (*NilType) String() string { return "()" }

type Integer int64

func (Integer) isValue()         {}
func (i Integer) String() string { return fmt.Sprintf("%d", int64(i)) }

type Float float64

func (Float) isValue()         {}
func (f Float) String() string { return fmt.Sprintf("%g", float64(f)) }

type String string

func (String) isValue()         {}
func (s String) String() string { return string(s) }

type Boolean bool

func (Boolean) isValue() {}
func (b Boolean) String() string {
	if b {
		return "#t"
	}
	return "#f"
}

type Sym struct {
	Name string
}

func (*Sym) isValue()         {}
func (s *Sym) String() string { return s.Name }

type Cons struct {
	Car Value
	Cdr Value
}

func (*Cons) isValue() {}
func (c *Cons) String() string {
	return "(" + consString(c) + ")"
}

func consString(c *Cons) string {
	r := c.Car.String()
	switch cdr := c.Cdr.(type) {
	case *NilType:
		return r
	case *Cons:
		return r + " " + consString(cdr)
	default:
		return r + " . " + cdr.String()
	}
}

type Primitive struct {
	Name string
	Fn   func(args []Value) (Value, error)
}

func (*Primitive) isValue() {}
func (p *Primitive) String() string {
	return fmt.Sprintf("#<builtin:%s>", p.Name)
}

type Vector struct {
	Items []Value
}

func (*Vector) isValue() {}

func (v *Vector) String() string {
	var b strings.Builder
	b.WriteString("#(")
	for i, item := range v.Items {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(item.String())
	}
	b.WriteString(")")
	return b.String()
}

type Closure struct {
	Env     *Env
	Params  []*Sym
	Body    []Value
	HasRest bool
	isAsync bool
}

func (*Closure) isValue()         {}
func (c *Closure) String() string { return "#<function>" }

type Macro struct {
	Env     *Env
	Params  []*Sym
	Body    []Value
	HasRest bool
}

type Future struct {
	result chan Value
	err    chan error
}

func (*Future) isValue()         {}
func (f *Future) String() string { return "#<future>" }

func NewFuture() *Future {
	return &Future{
		result: make(chan Value, 1),
		err:    make(chan error, 1),
	}
}

func (f *Future) Resolve(val Value, err error) {
	if err != nil {
		f.err <- err
	} else {
		f.result <- val
	}
}

func (f *Future) Await() (Value, error) {
	select {
	case val := <-f.result:
		return val, nil
	case err := <-f.err:
		return nil, err
	}
}

func (*Macro) isValue()         {}
func (m *Macro) String() string { return "#<macro>" }

func SliceToList(vals []Value) Value {
	if len(vals) == 0 {
		return Nil
	}
	return &Cons{Car: vals[0], Cdr: SliceToList(vals[1:])}
}

func ListToSlice(v Value) ([]Value, bool) {
	var result []Value
	for v != Nil {
		cons, ok := v.(*Cons)
		if !ok {
			return nil, false
		}
		result = append(result, cons.Car)
		v = cons.Cdr
	}
	return result, true
}

type SlotDef struct {
	Name    string
	Default Value
}

type ClassType struct {
	Name    string
	Parent  *ClassType
	Slots   []*SlotDef
	Methods map[string]*Closure
}

func (*ClassType) isValue()         {}
func (c *ClassType) String() string { return fmt.Sprintf("#<class:%s>", c.Name) }

type Instance struct {
	Class *ClassType
	Data  []Value
}

func (*Instance) isValue()         {}
func (i *Instance) String() string { return fmt.Sprintf("#<instance:%s>", i.Class.Name) }

type ReturnSignal struct {
	Value Value
}

func (r *ReturnSignal) Error() string  { return "<return>" }
func (*ReturnSignal) isValue()         {}
func (r *ReturnSignal) String() string { return r.Value.String() }

type HttpServer struct {
	Host    string
	Port    int
	Handler Value
}

func (*HttpServer) isValue()         {}
func (s *HttpServer) String() string { return fmt.Sprintf("#<http-server %s:%d>", s.Host, s.Port) }

type HttpRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    string
}

func (*HttpRequest) isValue()         {}
func (r *HttpRequest) String() string { return fmt.Sprintf("#<http-request %s %s>", r.Method, r.Path) }

type HttpResponse struct {
	Status  int
	Headers map[string]string
	Body    string
}

func (*HttpResponse) isValue()         {}
func (r *HttpResponse) String() string { return fmt.Sprintf("#<http-response %d>", r.Status) }

func IsTruthy(v Value) bool {
	switch val := v.(type) {
	case Boolean:
		return bool(val)
	case *NilType:
		return false
	default:
		return true
	}
}
