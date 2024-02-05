package flexrouter

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strconv"
	"testing"
)

func Test_Parts(t *testing.T) {
	testCases := []struct {
		Path     string
		Expected []string
	}{
		{"", nil},
		{"no-start-with-slash", nil},
		{"/", []string{"/"}},
		{"/abc", []string{"/abc"}},
		{"/ab/cde", []string{"/ab", "/cde"}},
		{"/a/bc/d/e/fgh", []string{"/a", "/bc", "/d", "/e", "/fgh"}},
		{"/abc/", []string{"/abc", "/"}},
		{"/a//b//", []string{"/a", "/", "/b", "/", "/"}},
	}

	for _, tc := range testCases {
		path := tc.Path
		expected := tc.Expected
		result := parts(path)
		if !slices.Equal(result, expected) {
			t.Fatalf(`Parts for path %s equal %+v instead of %+v`, path, result, expected)
		}
	}
}

func TestSpecPartNode(t *testing.T) {
	{
		part := "/literal"
		n := specPartNode(part, defaultParamNodeRegex)
		expected := node{
			Type: nodetype_literal,
			Part: part,
		}
		if !reflect.DeepEqual(n, expected) {
			t.Fatalf(`Node for path %s equals %+v instead of %+v`, part, n, expected)
		}
	}

	{
		part := "/{name:evaluator}"
		n := specPartNode(part, defaultParamNodeRegex)
		expected := node{
			Type:      nodetype_param,
			Part:      part,
			ParamName: "name",
			ParamEval: "evaluator",
		}
		if !reflect.DeepEqual(n, expected) {
			t.Fatalf(`Node for path %s equals %+v instead of %+v`, part, n, expected)
		}
	}
}

func makeHandler(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(msg))
	}
}

func Test_router_multipleMethods(t *testing.T) {
	rtr := NewRouter()

	rtr.AddRoute("/onlyget", Get(makeHandler("Get handler")))
	rtr.AddRoute("/any", Any(makeHandler("Any handler")))
	rtr.AddRoute("/getandpost", Get(makeHandler("Get handler")), Post(makeHandler("Post handler")))
	rtr.AddRoute("/getandpost1by1", Get(makeHandler("Get handler")))
	rtr.AddRoute("/getandpost1by1", Post(makeHandler("Get handler")))

	testCases := []struct {
		method             string
		path               string
		expectedStatusCode int
	}{
		{method: "GET", path: "/unknown/path", expectedStatusCode: 404},
		{method: "POST", path: "/unknown/path", expectedStatusCode: 404},
		{method: "GET", path: "/onlyget", expectedStatusCode: 200},
		{method: "POST", path: "/onlyget", expectedStatusCode: 405},
		{method: "GET", path: "/any", expectedStatusCode: 200},
		{method: "POST", path: "/any", expectedStatusCode: 200},
		{method: "GET", path: "/getandpost", expectedStatusCode: 200},
		{method: "POST", path: "/getandpost", expectedStatusCode: 200},
		{method: "GET", path: "/getandpost1by1", expectedStatusCode: 200},
		{method: "POST", path: "/getandpost1by1", expectedStatusCode: 200},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		wrec := httptest.NewRecorder()
		rtr.ServeHTTP(wrec, req)
		statusCode := wrec.Result().StatusCode
		if statusCode != tc.expectedStatusCode {
			t.Fatalf("Request %s %s Unexpected status code %d instead of %d", tc.method, tc.path, statusCode, tc.expectedStatusCode)
		}
	}
}

func Test_router_paramFuncs(t *testing.T) {
	rtr := NewRouter()

	rtr.SetParamFunc("uint", func(part string) (bool, interface{}) {
		value, err := strconv.Atoi(part)
		ok := err == nil && value >= 0 && strconv.Itoa(value) == part
		return ok, value
	})

	rtr.AddRoute("/user/{userId:uint}/name", Get(func(w http.ResponseWriter, r *http.Request) {
		userId := GetPathParams(r)["userId"].Value.(int)
		fmt.Fprintf(w, "userId:%d", userId)
	}))

	testCases := []struct {
		method             string
		path               string
		expectedStatusCode int
		expectedBody       string
	}{
		{method: "GET", path: "/user/123/name", expectedStatusCode: 200, expectedBody: "userId:123"},
		{method: "GET", path: "/user/-123/name", expectedStatusCode: 404, expectedBody: ""},
		{method: "GET", path: "/user/not-a-number/name", expectedStatusCode: 404, expectedBody: ""},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		wrec := httptest.NewRecorder()
		rtr.ServeHTTP(wrec, req)

		result := wrec.Result()
		statusCode := result.StatusCode
		if statusCode != tc.expectedStatusCode {
			t.Fatalf("Request %s %s Unexpected status code %d instead of %d", tc.method, tc.path, statusCode, tc.expectedStatusCode)
		}

		if tc.expectedBody != "" {
			bodyBuf, _ := io.ReadAll(result.Body)
			body := string(bodyBuf)
			if body != tc.expectedBody {
				t.Fatalf("Request %s %s Unexpected body %s instead of %s", tc.method, tc.path, body, tc.expectedBody)
			}
		}
	}
}
