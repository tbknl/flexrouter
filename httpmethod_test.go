package flexrouter

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_methods(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Test"))
	}

	testCases := []struct {
		handler        http.HandlerFunc
		method         string
		expectedResult bool
	}{
		// Any
		{handler: Any(handler), method: "GET", expectedResult: true},
		{handler: Any(handler), method: "HEAD", expectedResult: true},
		{handler: Any(handler), method: "POST", expectedResult: true},
		{handler: Any(handler), method: "PUT", expectedResult: true},
		{handler: Any(handler), method: "PATCH", expectedResult: true},
		{handler: Any(handler), method: "DELETE", expectedResult: true},
		{handler: Any(handler), method: "CONNECT", expectedResult: true},
		{handler: Any(handler), method: "OPTIONS", expectedResult: true},
		{handler: Any(handler), method: "TRACE", expectedResult: true},
		// Get
		{handler: Get(handler), method: "GET", expectedResult: true},
		{handler: Get(handler), method: "HEAD", expectedResult: false},
		// Head
		{handler: Head(handler), method: "HEAD", expectedResult: true},
		{handler: Head(handler), method: "POST", expectedResult: false},
		// Post
		{handler: Post(handler), method: "POST", expectedResult: true},
		{handler: Post(handler), method: "PUT", expectedResult: false},
		// Put
		{handler: Put(handler), method: "PUT", expectedResult: true},
		{handler: Put(handler), method: "PATCH", expectedResult: false},
		// Patch
		{handler: Patch(handler), method: "PATCH", expectedResult: true},
		{handler: Patch(handler), method: "DELETE", expectedResult: false},
		// Delete
		{handler: Delete(handler), method: "DELETE", expectedResult: true},
		{handler: Delete(handler), method: "CONNEXT", expectedResult: false},
		// Connect
		{handler: Connect(handler), method: "CONNECT", expectedResult: true},
		{handler: Connect(handler), method: "OPTIONS", expectedResult: false},
		// Options
		{handler: Options(handler), method: "OPTIONS", expectedResult: true},
		{handler: Options(handler), method: "TRACE", expectedResult: false},
		// Trace
		{handler: Trace(handler), method: "TRACE", expectedResult: true},
		{handler: Trace(handler), method: "GET", expectedResult: false},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, "/", nil)
		wrec := httptest.NewRecorder()
		tc.handler(wrec, req)
		body, err := io.ReadAll(wrec.Result().Body)
		hasResult := err == nil && string(body) == "Test"
		if hasResult != tc.expectedResult {
			fmt.Printf("%+v\n", body)
			t.Fatalf("Request %s on %v got result %v instead of %v", tc.method, tc.handler, hasResult, tc.expectedResult)
		}
	}
}
