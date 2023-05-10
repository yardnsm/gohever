package testutils

import (
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/dankinder/httpmock"
	"github.com/stretchr/testify/mock"
)

type FormData map[string]string

// MockedRequest is a wrapper around dankinder's httpmock library
type MockedRequest struct {
	method   string
	path     string
	response *httpmock.Response

	expectNot   bool        // Indicates this mock should *not* be called at all
	times       int         // The number of times this mock should be called. 0 means at least once.
	bodyMatcher interface{} // mock.argumentHandler is not exported :(
}

func NewMockedRequest(method string, path string) *MockedRequest {
	return &MockedRequest{
		method: method,
		path:   path,
		response: &httpmock.Response{
			Header: make(http.Header),
		},
		expectNot:   false,
		times:       0,
		bodyMatcher: mock.Anything,
	}
}

// Set the status code of the MockedRequest
func (m *MockedRequest) Status(statusCode int) *MockedRequest {
	m.response.Status = statusCode
	return m
}

// Set a header for the response
func (m *MockedRequest) Header(key, value string) *MockedRequest {
	m.response.Header.Add(key, value)
	return m
}

// Set the body of the MockedRequest to a file contents
func (m *MockedRequest) File(fileName string) *MockedRequest {
	data, err := os.ReadFile(fileName)

	if err != nil {
		panic(err)
	}

	m.response.Body = data

	return m
}

func (m *MockedRequest) Once() *MockedRequest {
	return m.Times(1)
}

func (m *MockedRequest) Times(times int) *MockedRequest {
	m.times = times
	return m
}

func (m *MockedRequest) ExpectNot() *MockedRequest {
	m.expectNot = true
	return m
}

// A mock.argumentHandler to match a body as given FormData
func (m *MockedRequest) MatchFormData(form FormData) *MockedRequest {
	m.bodyMatcher = mock.MatchedBy(func(body []byte) bool {
		unmarshalledBody, err := url.ParseQuery(string(body))
		if err != nil {
			return false
		}

		// url.Values returns a different type than FormData, so let's do a *very* naive thing
		// and sort og "flatten" the values
		bodyAsFormData := make(FormData)
		for k, v := range unmarshalledBody {
			bodyAsFormData[k] = v[0]
		}

		return reflect.DeepEqual(bodyAsFormData, form)
	})

	return m
}

// Setup a the MockedRequest on agiven httpmock.Handler
func (m *MockedRequest) Handle(t *testing.T, handler *httpmock.MockHandler) {
	args := []interface{}{
		m.method, m.path, m.bodyMatcher,
	}

	res := handler.On("Handle", args...).
		Return(*m.response)

	if m.times > 0 {
		res.Times(m.times)
	}

	if m.expectNot {
		res.Maybe()
		t.Cleanup(func() {
			handler.AssertNotCalled(t, "Handle", args...)
		})
	}
}
