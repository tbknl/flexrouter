package flexrouter

import (
	"context"
	"net/http"
	"regexp"
)

func parts(path string) []string {
	if len(path) == 0 || path[0] != '/' {
		return nil
	}
	pathParts := make([]string, 0, 20)
	curStart := 0
	for idx, ch := range path {
		if idx != curStart && ch == '/' {
			pathParts = append(pathParts, path[curStart:idx])
			curStart = idx
		}
	}
	pathParts = append(pathParts, path[curStart:len(path)])

	return pathParts
}

const (
	nodetype_literal = iota
	nodetype_param
)

type node struct {
	Type      int
	Part      string
	ParamName string
	ParamEval string
	SubNodes  []node
	Handlers  []http.Handler
}

var defaultParamNodeRegex = regexp.MustCompile("^/{([^:}]+):([^:}]+)}$")

func specPartNode(part string, paramRegex *regexp.Regexp) node {
	var matches []string
	if paramRegex != nil {
		matches = paramRegex.FindStringSubmatch(part)
	}

	if len(matches) > 0 {
		return node{
			Type:      nodetype_param,
			Part:      part,
			ParamName: matches[1],
			ParamEval: matches[2],
		}
	} else {
		return node{
			Type: nodetype_literal,
			Part: part,
		}
	}
}

func addPartsToTree(pathParts []string, handlers []http.Handler, nodes *[]node, paramRegex *regexp.Regexp) {
	part := pathParts[0]
	partNode := specPartNode(part, paramRegex)
	matchNode := &partNode

	for i := 0; i < len(*nodes); i++ {
		n := &(*nodes)[i]
		if n.Part == partNode.Part {
			matchNode = n
			break
		}
	}

	if len(pathParts) == 1 {
		matchNode.Handlers = append(matchNode.Handlers, handlers...)
	} else {
		addPartsToTree(pathParts[1:], handlers, &matchNode.SubNodes, paramRegex)
	}

	if matchNode == &partNode {
		*nodes = append(*nodes, partNode)
	}
}

type ParamFunc func(part string) (bool, interface{})
type ParamFuncMap map[string]ParamFunc

type Param struct {
	Name     string
	ValueRaw string
	Value    interface{}
}

func matchPartNode(part string, n *node, paramFuncs ParamFuncMap, param *Param) bool {
	switch n.Type {
	case nodetype_literal:
		return part == n.Part
	case nodetype_param:
		fn, exists := paramFuncs[n.ParamEval]
		if !exists {
			return false // Should only happens if paramFunc is not registerep
		}
		arg := part[1:]
		match := exists
		match, value := fn(arg)
		if match {
			*param = Param{
				Name:     n.ParamName,
				ValueRaw: arg,
				Value:    value,
			}
		}
		return match
	}
	return false
}

func findHandler(pathParts []string, params []Param, nodes []node, paramFuncs ParamFuncMap) []http.Handler {
	part := pathParts[0]

	for i := 0; i < len(nodes); i++ {
		n := &nodes[i]
		params[0] = Param{}
		if matchPartNode(part, n, paramFuncs, &params[0]) {
			if len(pathParts) == 1 {
				return n.Handlers
			} else {
				result := findHandler(pathParts[1:], params[1:], n.SubNodes, paramFuncs)
				if result != nil {
					return result
				}
			}
		}
	}

	return nil
}

type ParamMap map[string]Param

func matchPath(path string, nodes []node, paramFuncs ParamFuncMap) ([]http.Handler, ParamMap) {
	pathParts := parts(path)
	if pathParts == nil {
		return nil, nil // Invalid path provided.
	}

	params := make([]Param, len(pathParts))

	handlers := findHandler(pathParts, params, nodes, paramFuncs)
	if handlers != nil {
		pm := make(ParamMap, len(params))
		for _, param := range params {
			if param.Name != "" {
				pm[param.Name] = param
			}
		}
		return handlers, pm
	} else {
		return nil, nil
	}
}

type router struct {
	routes                  []node
	paramFuncs              ParamFuncMap
	ParamRegex              *regexp.Regexp
	NotFoundHandler         http.Handler
	MethodNotAllowedHandler http.Handler
}

var defaultNotFoundHandler = http.NotFoundHandler()
var defaultMethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Write([]byte("Method not allowed"))
})

func NewRouter() *router {
	return &router{
		routes:                  []node{},
		paramFuncs:              make(map[string]ParamFunc),
		ParamRegex:              defaultParamNodeRegex,
		NotFoundHandler:         defaultNotFoundHandler,
		MethodNotAllowedHandler: defaultMethodNotAllowedHandler,
	}
}

func (rtr *router) AddRoute(pathSpec string, handlers ...http.Handler) bool {
	if len(handlers) > 0 {
		pathParts := parts(pathSpec)
		if pathParts == nil {
			return false // Failure only happens when spec is invalid!
		}
		addPartsToTree(pathParts, handlers, &rtr.routes, rtr.ParamRegex)
		return true
	}
	return false
}

func (rtr *router) SetParamFunc(name string, fn ParamFunc) {
	if fn != nil {
		rtr.paramFuncs[name] = fn
	} else {
		delete(rtr.paramFuncs, name)
	}
}

type responseWriterWrapper struct {
	http.ResponseWriter
	written bool
}

func (w *responseWriterWrapper) WriteHeader(status int) {
	w.written = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	w.written = true
	return w.ResponseWriter.Write(b)
}

type pathParams struct{}

func GetPathParams(r *http.Request) ParamMap {
	return r.Context().Value(pathParams{}).(ParamMap)
}

func (rtr *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handlers, params := matchPath(r.URL.Path, rtr.routes, rtr.paramFuncs)
	if len(handlers) == 0 {
		if rtr.NotFoundHandler != nil {
			rtr.NotFoundHandler.ServeHTTP(w, r)
		}
	} else {
		rp := r.WithContext(context.WithValue(r.Context(), pathParams{}, params))
		ww := &responseWriterWrapper{w, false}
		for i := 0; i < len(handlers) && !ww.written; i++ {
			handlers[i].ServeHTTP(ww, rp)
		}
		if !ww.written {
			if rtr.MethodNotAllowedHandler != nil {
				rtr.MethodNotAllowedHandler.ServeHTTP(w, rp)
			}
		}
	}
}
