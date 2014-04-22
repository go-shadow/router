package router

import (
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Any callable function
type Handler interface{}

// Route holds individual route definitions
type Route struct {
	// The HTTP Method [GET, POST, PUT, DELETE]
	method string

	// The name of the route
	Name string

	// The regex pattern with named capture groups
	pattern string

	// The base pattern with regexes removed (used for generating routes)
	basePattern string

	// The Handlers
	Handlers []Handler
}

// Router is the main struct holding all the routes and requirements for requests
type Router struct {
	// An array of allowed extensions for requests
	validExtensions []string

	// Holds pointers to all routes by name
	routes map[string]*Route

	// Holds pointers to all routes by method
	routesByMethod map[string][]*Route

	// Holds all routes by method in groups of 15 with regexes
	regexesByMethod map[string][]*regexp.Regexp

	// Holds a pointer to the matched route
	matched *Route

	// Holds group name and prefix
	group map[string]string
}

// Private method called when adding a route. This handles convenience regex syntaxes and sets up regexes with named parameters
func newRoute(method string, name string, pattern string, handlers []Handler) (route *Route) {
	pattern = strings.Replace(pattern, "(int)", "([0-9]+)", -1)
	pattern = strings.Replace(pattern, "(alpha)", "([a-z]+)", -1)
	pattern = strings.Replace(pattern, "(alphanumeric)", "([a-z0-9]+)", -1)
	pattern = strings.Replace(pattern, "(slug)", "([a-z0-9-]+)", -1)
	pattern = strings.Replace(pattern, "(mongo)", "([0-9a-fA-F]{24})", -1)
	pattern = strings.Replace(pattern, "(md5)", "([0-9a-fA-F]{32})", -1)

	named := regexp.MustCompile(`:([a-zA-Z0-9_]+)`)
	namedRegex := regexp.MustCompile(`:([a-zA-Z0-9_]+)\(([^\)]+)\)`)

	// Create basic base pattern used for URL generation
	basePattern := namedRegex.ReplaceAllString(pattern, ":$1")

	if match := namedRegex.MatchString(pattern); match {
		pattern = namedRegex.ReplaceAllString(pattern, "(?P<$1>$2)")
	} else {
		pattern = named.ReplaceAllString(pattern, "(?P<$1>[^/]+)")
	}

	route = &Route{strings.ToUpper(method), name, pattern, basePattern, handlers}

	return
}

// Instantiates a new Router with basic settings
func New() *Router {
	router := new(Router)

	router.routes = make(map[string]*Route)

	// Allow requests with no extensions by default
	router.validExtensions = append(router.validExtensions, "")

	router.group = make(map[string]string, 2)

	router.routesByMethod = map[string][]*Route{
		"GET":    make([]*Route, 0),
		"POST":   make([]*Route, 0),
		"PUT":    make([]*Route, 0),
		"DELETE": make([]*Route, 0),
	}

	router.regexesByMethod = map[string][]*regexp.Regexp{
		"GET":    make([]*regexp.Regexp, 0),
		"POST":   make([]*regexp.Regexp, 0),
		"PUT":    make([]*regexp.Regexp, 0),
		"DELETE": make([]*regexp.Regexp, 0),
	}

	return router
}

// Generates a URL for the given Route
func (r *Route) URL(params ...interface{}) (url string) {
	url = r.basePattern

	for key, value := range params {
		if key%2 == 0 {
			continue
		}

		name := params[key-1].(string)

		switch v := value.(type) {
		case int:
			url = strings.Replace(url, ":"+name, strconv.Itoa(v), -1)
		case string:
			url = strings.Replace(url, ":"+name, v, -1)
		}
	}

	return
}

// Convenience method for adding GET routes
func (r *Router) Get(name string, pattern string, handlers ...Handler) *Route {
	return r.addRoute("GET", name, pattern, handlers...)
}

// Convenience method for adding POST routes
func (r *Router) Post(name string, pattern string, handlers ...Handler) *Route {
	return r.addRoute("POST", name, pattern, handlers)
}

// Convenience method for adding PUT routes
func (r *Router) Put(name string, pattern string, handlers ...Handler) *Route {
	return r.addRoute("PUT", name, pattern, handlers)
}

// Convenience method for adding DELETE routes
func (r *Router) Delete(name string, pattern string, handlers ...Handler) *Route {
	return r.addRoute("DELETE", name, pattern, handlers)
}

// Private method for setting all routes. This adds prefixes if we are withina  group
func (r *Router) addRoute(method string, name string, pattern string, handlers ...Handler) *Route {
	if r.group["name"] != "" {
		name = r.group["name"] + "_" + name
	}

	if r.group["prefix"] != "" {
		pattern = r.group["prefix"] + pattern
	}

	route := newRoute(method, name, pattern, handlers)

	r.routes[name] = route
	r.routesByMethod[method] = append(r.routesByMethod[method], route)

	return route
}

// Finds a route by name
func (r *Router) FindRoute(name string) (route *Route, found bool) {
	route, found = r.routes[name]

	return
}

// Sets a list of valid extensions allowed for requests (i.e. "json", "csv", "html")
func (r *Router) ValidExtensions(extensions ...string) *Router {
	r.validExtensions = extensions

	return r
}

// Compiles all regexes into groups of 15
func (r *Router) Compile() *Router {
	for method, _ := range r.routesByMethod {
		pattern := ""
		for i, route := range r.routesByMethod[method] {
			pattern += "(?P<" + route.Name + ">/)" + strings.TrimLeft(route.pattern, "/") + "|"

			if i > 0 && i%15 == 0 {
				pattern = "^(?:" + strings.TrimRight(pattern, "|") + ")$"
				r.regexesByMethod[method] = append(r.regexesByMethod[method], regexp.MustCompile(pattern))

				continue
			}
		}

		pattern = "^(?:" + strings.TrimRight(pattern, "|") + ")$"
		r.regexesByMethod[method] = append(r.regexesByMethod[method], regexp.MustCompile(pattern))
	}

	return r
}

// Checks to see if the request extension is valid
func (r *Router) extensionIsValid(ext string) bool {
	for _, valid := range r.validExtensions {
		if ext == valid {
			return true
		}
	}

	return false
}

// Loops through compiled routes to see if there is a matching route
func (r *Router) Dispatch(method string, path string) (*Route, map[string]interface{}) {
	regex := regexp.MustCompile(`\.([^\.]+)$`)
	params := make(map[string]interface{})
	var ext string
	var match []string

	if extMatch := regex.FindString(path); extMatch != "" {
		ext = strings.Replace(extMatch, ".", "", 1)
		path = regex.ReplaceAllLiteralString(path, "")
	}

	if !r.extensionIsValid(ext) {
		return nil, nil
	}

	for _, compiled := range r.regexesByMethod[method] {
		if match = compiled.FindStringSubmatch(path); match == nil {
			continue
		}

		for i, name := range compiled.SubexpNames() {
			paramLength := len(params)
			if i == 0 || match[i] == "" {
				if paramLength == 0 {
					continue
				}

				// All Params have been set. Empty matches means all params have been captured
				break
			}

			if paramLength == 0 {
				// Capture the name and set the ext so len(params) returns 1 on next loop
				r.matched, _ = r.FindRoute(name)

				params["ext"] = ext

				continue
			}

			if intValue, err := strconv.Atoi(match[i]); err == nil {
				params[name] = intValue

				continue
			}

			params[name] = match[i]
		}

		return r.matched, params
	}

	return nil, nil
}

// Generates a URL for a given route name
func (r *Router) URL(name string, params ...interface{}) string {
	route, exists := r.FindRoute(name)

	if exists {
		return route.URL(params...)
	}

	return ""
}

// @todo Don't use reflection
func (r *Router) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	route, _ := r.Dispatch(request.Method, request.URL.Path)

	var in = make([]reflect.Value, 2)
	in[0] = reflect.ValueOf(response)
	in[1] = reflect.ValueOf(request)

	for _, handler := range route.Handlers {
		reflect.ValueOf(handler).Call(in)
	}
}

// @todo Add support for handlers
func (r *Router) Group(prefix string, name string, fn func(*Router), handlers ...Handler) {
	r.group["prefix"] = strings.TrimRight(prefix, "/")
	r.group["name"] = strings.TrimRight(name, "_")

	fn(r)

	r.group["prefix"] = ""
	r.group["name"] = ""
}
