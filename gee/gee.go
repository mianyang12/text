package gee

import (
	"net/http"
)

type handlerFunc func(http.ResponseWriter, *http.Request)
type Engine struct {
	router map[string]handlerFunc
}

func New() *Engine {
	return &Engine{router: make(map[string]handlerFunc)}
}
func (engine *Engine) addRoute(method string, patten string, handler handlerFunc) {
	key := method + "-" + patten
	engine.router[key] = handler
}
func (engine *Engine) GET(patten string, handler handlerFunc) {
	engine.addRoute("GET", patten, handler)
}
func (engine *Engine) POST(patten string, handler handlerFunc) {
	engine.addRoute("POST", patten, handler)
}
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := req.Method + "-" + req.URL.Path
	if handler, ok := engine.router[key]; ok {
		handler(w, req)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 not found"))
	}
}
func (engine *Engine) Run(addr string) error {
	return http.ListenAndServe(addr, engine)
}
