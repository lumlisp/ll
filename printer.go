package main

import (
	"fmt"
	"io"
	"strings"
)

func Display(w io.Writer, v Value) {
	if v == nil {
		fmt.Fprint(w, "nil")
		return
	}
	switch val := v.(type) {
	case *NilType:
		fmt.Fprint(w, "()")
	case Boolean:
		if val {
			fmt.Fprint(w, "#t")
		} else {
			fmt.Fprint(w, "#f")
		}
	case String:
		fmt.Fprint(w, string(val))
	case *Sym:
		fmt.Fprint(w, val.Name)
	case *Cons:
		fmt.Fprint(w, "(")
		displayCons(w, val)
		fmt.Fprint(w, ")")
	case Integer:
		fmt.Fprint(w, int64(val))
	case Float:
		fmt.Fprint(w, float64(val))
	case *Primitive:
		fmt.Fprintf(w, "#<builtin:%s>", val.Name)
	case *Vector:
		fmt.Fprint(w, "#(")
		for i, item := range val.Items {
			if i > 0 {
				fmt.Fprint(w, " ")
			}
			Display(w, item)
		}
		fmt.Fprint(w, ")")
	case *Macro:
		fmt.Fprint(w, "#<macro>")
	case *Closure:
		fmt.Fprint(w, "#<function>")
	case *Future:
		fmt.Fprint(w, "#<future>")
	case *PdoConnection:
		fmt.Fprint(w, "#<pdo-connection>")
	case *WsServer:
		fmt.Fprintf(w, "#<ws-server %s:%d>", val.Host, val.Port)
	case *WsConn:
		fmt.Fprint(w, "#<ws-conn>")
	case *CgoLib:
		fmt.Fprintf(w, "#<cgo-lib:%s>", val.Name)
	default:
		fmt.Fprint(w, v.String())
	}
}

func displayCons(w io.Writer, c *Cons) {
	Display(w, c.Car)
	switch cdr := c.Cdr.(type) {
	case *NilType:
		return
	case *Cons:
		fmt.Fprint(w, " ")
		displayCons(w, cdr)
	default:
		fmt.Fprint(w, " . ")
		Display(w, cdr)
	}
}

func FormatValue(v Value) string {
	var b strings.Builder
	Display(&b, v)
	return b.String()
}

func Sprint(v Value) string {
	return mustSprint(v)
}

func mustSprint(v Value) string {
	switch val := v.(type) {
	case *NilType:
		return "()"
	case Boolean:
		if val {
			return "#t"
		}
		return "#f"
	case String:
		return quoteString(string(val))
	case *Sym:
		return val.Name
	case *Cons:
		return "(" + sprintCons(val) + ")"
	case Integer:
		return fmt.Sprintf("%d", int64(val))
	case Float:
		return fmt.Sprintf("%g", float64(val))
	case *Primitive:
		return fmt.Sprintf("#<builtin:%s>", val.Name)
	case *Vector:
		var b strings.Builder
		b.WriteString("#(")
		for i, item := range val.Items {
			if i > 0 {
				b.WriteString(" ")
			}
			b.WriteString(mustSprint(item))
		}
		b.WriteString(")")
		return b.String()
	case *Macro:
		return "#<macro>"
	case *Closure:
		return "#<function>"
	case *Future:
		return "#<future>"
	case *PdoConnection:
		return "#<pdo-connection>"
	case *WsServer:
		return fmt.Sprintf("#<ws-server %s:%d>", val.Host, val.Port)
	case *WsConn:
		return "#<ws-conn>"
	case *CgoLib:
		return fmt.Sprintf("#<cgo-lib:%s>", val.Name)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func sprintCons(c *Cons) string {
	r := mustSprint(c.Car)
	switch cdr := c.Cdr.(type) {
	case *NilType:
		return r
	case *Cons:
		return r + " " + sprintCons(cdr)
	default:
		return r + " . " + mustSprint(cdr)
	}
}

func quoteString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
