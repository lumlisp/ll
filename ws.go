package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

func (e *Eval) builtinWsCreateServer(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("ws/create-server requires 2 arguments (host port)")
	}
	host, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("ws/create-server: host must be a string")
	}
	port, ok := args[1].(Integer)
	if !ok {
		return nil, fmt.Errorf("ws/create-server: port must be an integer")
	}
	return &WsServer{
		Host:    string(host),
		Port:    int(port),
		Handler: Nil,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}, nil
}

func (e *Eval) builtinWsSetHandler(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("ws/set-handler requires 2 arguments (server handler)")
	}
	server, ok := args[0].(*WsServer)
	if !ok {
		return nil, fmt.Errorf("ws/set-handler: first argument must be a ws-server")
	}
	switch args[1].(type) {
	case *Closure, *Primitive:
		server.Handler = args[1]
	default:
		return nil, fmt.Errorf("ws/set-handler: second argument must be a function")
	}
	return Nil, nil
}

func (e *Eval) builtinWsStartServer(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ws/start-server requires 1 argument")
	}
	server, ok := args[0].(*WsServer)
	if !ok {
		return nil, fmt.Errorf("ws/start-server: argument must be a ws-server")
	}
	if server.Handler == nil || server.Handler == Nil {
		return nil, fmt.Errorf("ws/start-server: no handler set")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := server.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			wrapped := &WsConn{Conn: conn}
			_, err = e.Apply(server.Handler, []Value{wrapped, String(string(msg))})
			if err != nil {
				break
			}
		}
	})

	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
	srv := &http.Server{Addr: addr, Handler: mux}

	f := NewFuture()
	go func() {
		fmt.Fprintf(e.w, "WebSocket server listening on %s\n", addr)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			f.Resolve(Nil, err)
			return
		}
		f.Resolve(Nil, nil)
	}()
	return f, nil
}

func (e *Eval) builtinWsConnect(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ws/connect requires 1 argument (url)")
	}
	url, ok := args[0].(String)
	if !ok {
		return nil, fmt.Errorf("ws/connect: url must be a string")
	}
	conn, _, err := websocket.DefaultDialer.Dial(string(url), nil)
	if err != nil {
		return nil, fmt.Errorf("ws/connect: %v", err)
	}
	return &WsConn{Conn: conn}, nil
}

func (e *Eval) builtinWsSend(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("ws/send requires 2 arguments (conn message)")
	}
	wc, ok := args[0].(*WsConn)
	if !ok {
		return nil, fmt.Errorf("ws/send: first argument must be a ws-conn")
	}
	msg, ok := args[1].(String)
	if !ok {
		return nil, fmt.Errorf("ws/send: message must be a string")
	}
	if err := wc.Conn.WriteMessage(websocket.TextMessage, []byte(string(msg))); err != nil {
		return nil, fmt.Errorf("ws/send: %v", err)
	}
	return Nil, nil
}

func (e *Eval) builtinWsReceive(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ws/receive requires 1 argument (conn)")
	}
	wc, ok := args[0].(*WsConn)
	if !ok {
		return nil, fmt.Errorf("ws/receive: first argument must be a ws-conn")
	}
	_, msg, err := wc.Conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("ws/receive: %v", err)
	}
	return String(string(msg)), nil
}

func (e *Eval) builtinWsClose(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ws/close requires 1 argument")
	}
	wc, ok := args[0].(*WsConn)
	if !ok {
		return nil, fmt.Errorf("ws/close: argument must be a ws-conn")
	}
	if err := wc.Conn.Close(); err != nil {
		return nil, fmt.Errorf("ws/close: %v", err)
	}
	return Nil, nil
}
