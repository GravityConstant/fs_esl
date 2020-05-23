/*
 * 取自golang的net/http包，略有改动
 * 必须以/开头，/结尾；如：/ivr/，因为删除了自动完善路径的方法
 * 可以携带参数，如：/ivr/:frist/的路由对应handlefunc的/ivr/true
 * 参数都会被解析成string类型，遇到第一个:表示后面均为参数
 */
package dialplan

import (
	"sort"
	"strings"
	"sync"

	"github.com/0x19/goesl"
)

type Handler interface {
	ServeFreeswitch(Response, *Request)
}

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as HTTP handlers. If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// Handler that calls f.
type HandlerFunc func(Response, *Request)

// ServeHTTP calls f(w, r).
func (f HandlerFunc) ServeFreeswitch(w Response, r *Request) {
	f(w, r)
}

// Helper handlers

// Error replies to the request with the specified error message and HTTP code.
// It does not otherwise end the request; the caller should ensure no further
// writes are done to w.
// The error message should be plain text.
func Error(w Response, error string, code int) {
	goesl.Error("error: %s, %d", error, code)
	switch code {
	case 404:
		w.Conn.Execute("hangup", "NO_ROUTE_DESTINATION", false)
	default:
		w.Conn.Execute("hangup", "", false)
	}

}

// NotFound replies to the request with an HTTP 404 not found error.
func NotFound(w Response, r *Request) { Error(w, "404 page not found", 404) }

// NotFoundHandler returns a simple request handler
// that replies to each request with a ``404 page not found'' reply.
func NotFoundHandler() Handler { return HandlerFunc(NotFound) }

type ServeMux struct {
	mu    sync.RWMutex
	m     map[string]muxEntry
	es    []muxEntry // slice of entries sorted from longest to shortest.
	hosts bool       // whether any patterns contain hostnames
}

type muxEntry struct {
	h       Handler
	pattern string
	params  []string
}

// NewServeMux allocates and returns a new ServeMux.
func NewServeMux() *ServeMux { return new(ServeMux) }

// DefaultServeMux is the default ServeMux used by Serve.
var DefaultServeMux = &defaultServeMux

var defaultServeMux ServeMux

// Find a handler on a handler map given a path string.
// Most-specific (longest) pattern wins.
func (mux *ServeMux) match(path string) (h Handler, pattern string) {
	// Check for exact match first.
	v, ok := mux.m[path]
	if ok {
		return v.h, v.pattern
	}

	// Check for longest valid match.  mux.es contains all patterns
	// that end in / sorted from longest to shortest.
	for _, e := range mux.es {
		if strings.HasPrefix(path, e.pattern) {
			return e.h, e.pattern
		}
	}
	return nil, ""
}

// Handler returns the handler to use for the given request,
// consulting r.Method, r.Host, and r.URL.Path. It always returns
// a non-nil handler. If the path is not in its canonical form, the
// handler will be an internally-generated handler that redirects
// to the canonical path. If the host contains a port, it is ignored
// when matching handlers.
//
// The path and host are used unchanged for CONNECT requests.
//
// Handler also returns the registered pattern that matches the
// request or, in the case of internally-generated redirects,
// the pattern that will match after following the redirect.
//
// If there is no registered handler that applies to the request,
// Handler returns a ``page not found'' handler and an empty pattern.
func (mux *ServeMux) Handler(r *Request) (h Handler, pattern string) {
	return mux.handler(r.Header["variable_socket_host"], r.URL.Path)
}

// handler is the main implementation of Handler.
// The path is known to be in canonical form, except for CONNECT methods.
func (mux *ServeMux) handler(host, path string) (h Handler, pattern string) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	// Host-specific pattern takes precedence over generic ones
	if mux.hosts {
		h, pattern = mux.match(host + path)
	}
	if h == nil {
		h, pattern = mux.match(path)
	}
	if h == nil {
		h, pattern = NotFoundHandler(), ""
	}
	return
}

// ServeHTTP dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (mux *ServeMux) ServeFreeswitch(w Response, r *Request) {
	if len(r.URL.Path) == 0 {
		goesl.Error("request uri is empty.")
		return
	}
	h, pattern := mux.Handler(r)
	ps := strings.Split(strings.TrimLeft(r.URL.Path, pattern), "/")
	for i := 0; i < len(ps); i++ {
		if len(ps[i]) > 0 {
			r.Params[mux.m[pattern].params[i]] = ps[i]
		}
	}
	goesl.Debug("matched handler: %#v\n", h)
	h.ServeFreeswitch(w, r)
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if pattern == "" {
		panic("http: invalid pattern")
	}
	if handler == nil {
		panic("http: nil handler")
	}
	if _, exist := mux.m[pattern]; exist {
		panic("http: multiple registrations for " + pattern)
	}

	// exist ':' is param
	params := make([]string, 0)
	if strings.Contains(pattern, ":") {
		paths := strings.Split(pattern, "/")
		for _, p := range paths {
			if strings.Contains(p, ":") {
				params = append(params, p[1:])
			}
		}
		pattern = pattern[:strings.Index(pattern, ":")]
	}

	if mux.m == nil {
		mux.m = make(map[string]muxEntry)
	}
	e := muxEntry{h: handler, pattern: pattern, params: params}
	mux.m[pattern] = e
	if pattern[len(pattern)-1] == '/' {
		mux.es = appendSorted(mux.es, e)
	}

	if pattern[0] != '/' {
		mux.hosts = true
	}
}

func appendSorted(es []muxEntry, e muxEntry) []muxEntry {
	n := len(es)
	i := sort.Search(n, func(i int) bool {
		return len(es[i].pattern) < len(e.pattern)
	})
	if i == n {
		return append(es, e)
	}
	// we now know that i points at where we want to insert
	es = append(es, muxEntry{}) // try to grow the slice in place, any entry works.
	copy(es[i+1:], es[i:])      // Move shorter entries down
	es[i] = e
	return es
}

// HandleFunc registers the handler function for the given pattern.
func (mux *ServeMux) HandleFunc(pattern string, handler func(Response, *Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	mux.Handle(pattern, HandlerFunc(handler))
}

func HandleFunc(pattern string, handler func(Response, *Request)) {
	DefaultServeMux.HandleFunc(pattern, handler)
}
