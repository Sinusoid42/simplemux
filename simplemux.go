package simplemux

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type MiddlewareFunc func(http.Handler) http.Handler

func (router *Router) use(middleware MiddlewareFunc) {
	router.middleware = append(router.middleware, middleware)
}

func (mux *Multiplexer) Use(middleware MiddlewareFunc) {
	mux.router.use(middleware)
}

type route_segment struct {
	is_var   bool
	sub_path string
}

type Route struct {
	segments    []route_segment
	method      string
	Handler     http.HandlerFunc
	contentType string // New field to handle content type
}

type RouterMetricCollector struct {
	num_requests uint64
	uptime       uint64
}

type Router struct {
	routes []Route

	not_found_route     Route
	has_not_found_route bool
	// metrics collector
	metrics_collector *RouterMetricCollector

	middleware []MiddlewareFunc
}

func (mux *Multiplexer) AddGetRoute(route string, handler http.HandlerFunc) {
	mux.AddRoute("GET "+route, handler, "")
}

func (mux *Multiplexer) AddPostRoute(route string, handler http.HandlerFunc) {
	mux.AddRoute("POST "+route, handler, "")
}

func (mux *Multiplexer) AddPutRoute(route string, handler http.HandlerFunc) {
	mux.AddRoute("PUT "+route, handler, "")
}

func (mux *Multiplexer) AddDeleteRoute(route string, handler http.HandlerFunc) {
	mux.AddRoute("DELETE "+route, handler, "")
}

func GetPathParams(r *http.Request) map[string]string {
	if params, ok := r.Context().Value("params").(map[string]string); ok {
		return params
	}
	return nil
}
func GetQueryParams(r *http.Request) map[string][]string {
	return r.URL.Query()
}

func (router *Router) ServeStatic(pathPrefix, fileDir string) {
	fileServer := http.FileServer(http.Dir(fileDir))
	staticHandler := http.StripPrefix(pathPrefix, fileServer)
	router.add_route("GET "+pathPrefix+"/*", staticHandler.ServeHTTP, "")
}

type Multiplexer struct {
	server         *http.Server
	router         *Router
	stopChannel    chan struct{}
	wg             *sync.WaitGroup
	upgradeHandler http.HandlerFunc
}

type Mux_config struct {
	Addr string // Address and port to listen on, e.g., "localhost:8080"
	tls  bool   // Whether to use HTTPS
	Cert string // Path to TLS certificate file
	Key  string // Path to TLS key file
}

func (mux *Multiplexer) AddRoute(methodRoute string, handler http.HandlerFunc, contentType string) {
	mux.router.add_route(methodRoute, handler, contentType)
}

func (mux *Multiplexer) Redirect(methodRoute string, redirectUrl string) {
	redirectHandler := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectUrl, http.StatusFound)
	}
	mux.router.add_route(methodRoute, redirectHandler, "")
}

func Generate_mulitplexer() *Multiplexer {
	return empty_mulitplexer()
}

func empty_mulitplexer() *Multiplexer {
	mux := &Multiplexer{
		server:      nil,
		router:      &Router{},
		stopChannel: nil,
		wg:          &sync.WaitGroup{},
	}

	return mux
}

func (router *Router) add_route(methodRoute string, handler http.HandlerFunc, contentType string) {
	parts := strings.SplitN(methodRoute, " ", 2)

	var method, pattern string
	if strings.Contains(parts[0], "/") {
		method = ""
		pattern = parts[0]
	} else if len(parts) == 2 {
		method = parts[0]
		pattern = parts[1]
	} else {
		fmt.Printf("Invalid methodRoute format: %s\n", methodRoute)
		return
	}

	segments := parsePattern(pattern)

	route := Route{
		segments:    segments,
		method:      method,
		Handler:     handler,
		contentType: contentType,
	}
	router.routes = append(router.routes, route)
}

func (router *Router) add_not_found_route(handler http.HandlerFunc) {

	if router.has_not_found_route {
		fmt.Println("Not found route already exists")
		return
	}

	if router.not_found_route.segments != nil {
		fmt.Println("Not found route already exists")
		return
	}

	if handler == nil {
		fmt.Println("Handler is nil")
		router.has_not_found_route = false

		return
	}

	router.not_found_route = Route{
		segments: nil,
		method:   "",
		Handler:  handler,
	}

	router.has_not_found_route = true
}

func parsePattern(pattern string) []route_segment {
	var segments []route_segment

	string_segments := strings.Split(pattern, "/")

	for i := 0; i < len(string_segments); i++ {
		segment := string_segments[i]
		//for _, segment := range string_segments {

		if segment == "" && i > 0 {
			segments = append(segments, route_segment{
				is_var:   false,
				sub_path: "/",
			})
			break
		}

		if segment == "" {
			continue
		}

		isVar := strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
		if isVar {
			segments = append(segments, route_segment{
				is_var:   isVar,
				sub_path: strings.Trim(segment, "{}"),
			})
		} else {
			segments = append(segments, route_segment{
				is_var:   false,
				sub_path: segment,
			})
		}

	}
	return segments
}

func (router *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var handler http.Handler
	hasRoute := false

	for _, route := range router.routes {
		if params, ok := match(route.segments, req.URL.Path); ok {
			if route.method != "" && route.method != req.Method {
				continue
			}

			if route.contentType != "" && route.contentType != req.Header.Get("Content-Type") {
				continue
			}

			hasRoute = true
			req = addParamsToRequest(req, params)
			handler = route.Handler
			break
		}
	}

	if handler == nil {
		if hasRoute {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if router.has_not_found_route {
			handler = router.not_found_route.Handler
		} else {
			http.NotFound(w, req)
			return
		}
	}

	for i := len(router.middleware) - 1; i >= 0; i-- {
		handler = router.middleware[i](handler)
	}

	handler.ServeHTTP(w, req)
}

func splitPath(path string) []string {
	// Split the path into segments on '/'
	rawSegments := strings.Split(path, "/")

	// Create a slice to hold the non-empty segments
	var segments []string

	// Iterate through the raw segments, ignoring any empty segments
	for _, segment := range rawSegments {
		if len(segment) > 0 {
			segments = append(segments, segment)
		}
	}

	// Check if the original path had a trailing slash
	if !strings.HasPrefix(path, "/") && len(segments) == 0 {
		segments = append(segments, "/")
	}

	if len(segments) == 0 {
		segments = append(segments, "/")
	}
	return segments
}

func match(segments []route_segment, path string) (map[string]string, bool) {
	pathSegments := splitPath(path)
	if len(segments) != len(pathSegments) {
		return nil, false
	}
	params := make(map[string]string)
	for i := 0; i < len(segments); i++ {
		if segments[i].is_var {
			params[segments[i].sub_path] = pathSegments[i]
		} else if segments[i].sub_path != pathSegments[i] {
			return nil, false
		}

	}
	return params, true
}

func addParamsToRequest(req *http.Request, params map[string]string) *http.Request {
	ctx := context.WithValue(req.Context(), "params", params)
	return req.WithContext(ctx)
}

func (sm *Multiplexer) Start(config *Mux_config) {
	sm.stopChannel = make(chan struct{})

	//sm.router = &DBRouter{}
	sm.server = &http.Server{
		Addr:    config.Addr,
		Handler: sm.router,
	}

	if config.Key != "" && config.Cert != "" {

		_, eKey := os.Stat(config.Key)
		_, eCert := os.Stat(config.Cert)
		if eKey == nil && eCert == nil {
			config.tls = true
		} else {
			fmt.Println("Key or Cert file not found")
		}
	}

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		var err error
		if config.tls {
			err = sm.server.ListenAndServeTLS(config.Cert, config.Key)
		} else {
			err = sm.server.ListenAndServe()
		}
		if err != http.ErrServerClosed || err != nil {
			fmt.Println("Server failed: %v", err)
			panic(err)
		}
	}()

	fmt.Println("Server started on %s", config.Addr)

}
func (sm *Multiplexer) Stop() {
	close(sm.stopChannel)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sm.server.Shutdown(ctx); err != nil {
		fmt.Println("Server Shutdown Failed:%+v", err)
	}
	sm.wg.Wait()
	fmt.Println("Server gracefully stopped")
}

func (sm *Multiplexer) Restart(config *Mux_config) {
	sm.Stop()
	sm.Start(config)
}

func (mux *Multiplexer) Wait() error {
	if mux.wg != nil {
		mux.wg.Wait()
	}
	return nil
}
