package testutils

import (
	"testing"

	"github.com/dankinder/httpmock"
)

type MockServer struct {
	handler *httpmock.MockHandler
	server  *httpmock.Server
	t *testing.T
}

// Creates a new mock server, which wont start until a test is attached
func NewMockServer() *MockServer {
	mockServer := &MockServer{}

	mockServer.handler = &httpmock.MockHandler{}
	mockServer.server = httpmock.NewUnstartedServer(mockServer.handler)

	return mockServer
}

// Setup the server for a test, will start the server and close is when the test ends
func (m *MockServer) SetupTest(t *testing.T) *MockServer {
	m.t = t

	m.server.Start()

	t.Cleanup(func() {
		m.server.Close()
		m.handler.AssertExpectations(t)
	})

	return m;
}

// Setup a MockedRequest on thre server
func (m *MockServer) Mock(r *MockedRequest) *MockServer {
	r.Handle(m.t, m.handler)
	return m;
}

// Returns the URL of the mocked server
func (m *MockServer) URL() string {
	return m.server.URL();
}
