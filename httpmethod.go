package flexrouter

import "net/http"

type HttpMethod string

func (hm HttpMethod) MethodHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == string(hm) {
			handler(w, r)
		}
	}
}

type Any = http.HandlerFunc

var Get = HttpMethod(http.MethodGet).MethodHandler
var Head = HttpMethod(http.MethodHead).MethodHandler
var Post = HttpMethod(http.MethodPost).MethodHandler
var Put = HttpMethod(http.MethodPut).MethodHandler
var Patch = HttpMethod(http.MethodPatch).MethodHandler
var Delete = HttpMethod(http.MethodDelete).MethodHandler
var Connect = HttpMethod(http.MethodConnect).MethodHandler
var Options = HttpMethod(http.MethodOptions).MethodHandler
var Trace = HttpMethod(http.MethodTrace).MethodHandler
