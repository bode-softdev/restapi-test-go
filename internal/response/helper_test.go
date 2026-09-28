package response

import (
	"bytes"
	"net/http"
	"net/http/httptest"
)

func newRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func newRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, bytes.NewBufferString("{}"))
}
