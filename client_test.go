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
	isError      bool
	statusCode   int
	responseBody string
}

func (c *httpClientMock) Do(req *http.Request) (*http.Response, error) {
	if c.isError {
		return nil, fmt.Errorf("error doing request")
	}

	response := httptest.NewRecorder()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(c.statusCode)
		echoBody, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		w.Write(echoBody)
	})

	h.ServeHTTP(response, req)

	return response.Result(), nil
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
			errMsg: `failed to get API response: error doing request`,
			httpClient: &httpClientMock{
				isError: true,
			},
		},
		"response code is 400": {
			url:         "http://localhost",
			method:      http.MethodGet,
			requestBody: strings.NewReader(""),
			errMsg:      "response code is 400",
			httpClient: &httpClientMock{
				statusCode: 400,
			},
		},
		"json error": {
			url:         "http://localhost",
			method:      http.MethodGet,
			requestBody: strings.NewReader(""),
			response:    struct{}{},
			errMsg:      "failed to decode JSON response: EOF",
			httpClient: &httpClientMock{
				statusCode: 200,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := NewClient(
				"",
				tt.httpClient,
			)

			err := client.doRequest(context.Background(), tt.method, tt.url, tt.requestBody, tt.response)
			if err == nil {
				t.Fatal("expected error, got none")
			}

			if err.Error() != tt.errMsg {
				t.Errorf("error message missmatch: want `%s`, got `%s`", tt.errMsg, err.Error())
			}
		})
	}
}

func TestClient_doRequest_success(t *testing.T) {
	t.Parallel()

	type testResponse struct {
		Success bool `json:"success"`
	}

	want := &testResponse{true}
	// todo test method
	// todo test url
	// todo test ctx

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse := `{
				"success": true
			}`
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jsonResponse))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, http.DefaultClient)

	got := &testResponse{}
	err := client.doRequest(context.Background(), http.MethodGet, "", nil, got)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want `%v`, got `%v`", want, got)
	}
}
