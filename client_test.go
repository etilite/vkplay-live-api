package vkplayliveapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type httpClientMock struct {
	doMock func(req *http.Request) (*http.Response, error)
}

func (c *httpClientMock) Do(req *http.Request) (*http.Response, error) {
	return c.doMock(req)
}

func TestClient_doRequest_errors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		url         string
		method      string
		requestBody io.Reader
		response    any
		errMsg      string
		httpClient  HTTPClient
	}{
		"failed build request": {
			url:    ":foo",
			method: http.MethodGet,
			errMsg: "failed to build request: parse \":foo\": missing protocol scheme",
		},
		"failed doing request": {
			url:    "http://localhost",
			method: http.MethodGet,
			errMsg: `failed to get API response: failed to do request`,
			httpClient: &httpClientMock{
				doMock: func(req *http.Request) (*http.Response, error) {
					return nil, fmt.Errorf("failed to do request")
				},
			},
		},
		"response code is 400": {
			url:    "http://localhost",
			method: http.MethodGet,
			errMsg: "response code is 400",
			httpClient: &httpClientMock{
				doMock: func(req *http.Request) (*http.Response, error) {
					response := httptest.NewRecorder()
					h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusBadRequest)
					})

					h.ServeHTTP(response, req)

					return response.Result(), nil
				},
			},
		},
		"json error": {
			url:      "http://localhost",
			method:   http.MethodGet,
			response: struct{}{},
			errMsg:   "failed to decode JSON response: EOF",
			httpClient: &httpClientMock{
				doMock: func(req *http.Request) (*http.Response, error) {
					response := httptest.NewRecorder()
					h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						_, err := w.Write([]byte(""))
						if err != nil {
							t.Error("failed to write response")
						}
					})

					h.ServeHTTP(response, req)

					return response.Result(), nil
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := newClient(
				"",
				tt.httpClient,
			)

			err := c.doRequest(context.Background(), tt.method, tt.url, tt.requestBody, tt.response)
			if err == nil {
				t.Fatal("expected error, got none")
			}

			if err.Error() != tt.errMsg {
				t.Errorf("error message missmatch: want `%s`, got `%s`", tt.errMsg, err.Error())
			}
		})
	}
}

func TestClient_doRequest_arguments(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx    context.Context
		method string
		url    string
		body   string
	}
	var got args

	httpMock := &httpClientMock{
		doMock: func(req *http.Request) (*http.Response, error) {
			response := httptest.NewRecorder()
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("{}"))
				if err != nil {
					t.Error("failed to write response")
				}
			})
			h.ServeHTTP(response, req)

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Error("failed to read")
			}

			got = args{
				ctx:    req.Context(),
				method: req.Method,
				url:    req.URL.String(),
				body:   string(body),
			}

			return response.Result(), nil
		},
	}

	c := newClient(
		"http://localhost",
		httpMock,
	)

	type CtxKey string
	uniqKey := CtxKey("test")

	ctx := context.WithValue(context.Background(), uniqKey, "unique-ctx-value")
	want := args{
		ctx:    ctx,
		method: http.MethodGet,
		url:    "http://localhost/url",
		body:   `{"unique-value":"unique-value"}`,
	}

	err := c.doRequest(want.ctx, want.method, "/url", strings.NewReader(want.body), nil)
	if err != nil {
		t.Fatal("failed to do request")
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want `%v`, got `%v`", want, got)
	}
}

func TestClient_doRequest_success(t *testing.T) {
	t.Parallel()

	type testResponse struct {
		Success bool `json:"success"`
	}

	want := &testResponse{true}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse := `{
				"success": true
			}`
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jsonResponse))
	}))
	defer srv.Close()

	c := newClient(srv.URL, http.DefaultClient)

	got := &testResponse{}
	err := c.doRequest(context.Background(), http.MethodGet, "", nil, got)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want `%v`, got `%v`", want, got)
	}
}
