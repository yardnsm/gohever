package gohever

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yardnsm/gohever/internal/mocks"
	"github.com/yardnsm/gohever/testutils"
)

func TestWrapAuthenticated(t *testing.T) {
	// A struct used for testing the requestHandler result
	type testStruct struct{ key string }

	setupTest := func(t *testing.T) (*Client, *mocks.AuthInterface, *mocks.RequestHandler[testStruct]) {
		client := NewClient(FlavorHvr, Config{})

		authMock := mocks.NewAuthInterface(t)
		requestHandlerMock := mocks.NewRequestHandler[testStruct](t)

		// Setup mocks
		client.Auth = authMock

		// Be authenticated by default
		client.isAuthenticated = true

		return client, authMock, requestHandlerMock
	}

	t.Run("should authenticate by default when not authenticated", func(t *testing.T) {
		client, authMock, handler := setupTest(t)

		client.isAuthenticated = false

		authMock.On("Authenticate").Once().Return(nil)
		handler.On("Execute").Return(nil, nil) // Handler does nothing

		wrapAuthenticated(client, handler.Execute)()
	})

	t.Run("should not authenticate when the handler return data", func(t *testing.T) {
		client, authMock, handler := setupTest(t)

		// Handler returns something, and no error
		handler.On("Execute").Return(&testStruct{key: "value"}, nil)

		data, err := wrapAuthenticated(client, handler.Execute)()

		authMock.AssertNumberOfCalls(t, "Authenticate", 0)

		assert.Equal(t, data, &testStruct{key: "value"})
		assert.Equal(t, err, nil)
	})

	t.Run("should not authenticate when the handler fails with non-auth reason", func(t *testing.T) {
		client, authMock, handler := setupTest(t)

		// Handler returns something, and an error (something random for the sake of it)
		handler.On("Execute").Return(nil, ErrRedirectIsNotAllowed)

		_, err := wrapAuthenticated(client, handler.Execute)()

		authMock.AssertNumberOfCalls(t, "Authenticate", 0)

		assert.ErrorIs(t, err, ErrRedirectIsNotAllowed)
	})

	t.Run("should authenticate when the handler fails with auth reson", func(t *testing.T) {
		client, authMock, handler := setupTest(t)

		// Handler returns an auth error
		handler.On("Execute").Once().Return(nil, ErrNotAuthenticated)
		handler.On("Execute").Once().Return(&testStruct{key: "value"}, nil)

		authMock.On("Authenticate").Once().Return(nil)

		data, err := wrapAuthenticated(client, handler.Execute)()

		assert.Equal(t, data, &testStruct{key: "value"})
		assert.Equal(t, err, nil)
	})

	t.Run("should return an error when authentication fails", func(t *testing.T) {
		client, authMock, handler := setupTest(t)

		// Handler returns an auth error
		handler.On("Execute").Once().Return(nil, ErrNotAuthenticated)

		authMock.On("Authenticate").Once().Return(ErrAuthenticatedFailed)

		_, err := wrapAuthenticated(client, handler.Execute)()

		assert.ErrorIs(t, err, ErrAuthenticatedFailed)
	})
}

func TestGetConfig(t *testing.T) {
	client := SetupTestClient(t, TestClientConfig{
		Mocks: []*testutils.MockedRequest{
			testutils.NewMockedRequest("GET", "/signin.aspx?bs=1").Status(200).File("testdata/auth_get_config.html"),
		},
	})

	auth := newAuth(client)

	cfg, err := auth.getConfig()

	assert.Equal(t, err, nil)
	assert.Equal(t, cfg, &authenticationConfig{
		formData: formData{
			"bs":            "1",
			"cn":            "12341234134",
			"emailRestore":  "",
			"email_loc":     "",
			"oMode":         "login",
			"redirect":      "",
			"reffer":        "",
			"tmpl_filename": "signin_hvr",
			"tz":            "TestUsername", // Taken from the default HeverCredentials defined in ./client_test.go
			"password":      "TestPassword", // Taken from the default HeverCredentials defined in ./client_test.go
		},
		verifyPixelUrl: "acmplt.asmx/logo?t=1234123412341",
	})
}

func TestSendVerifyPixel(t *testing.T) {
	client := SetupTestClient(t, TestClientConfig{
		Mocks: []*testutils.MockedRequest{
			testutils.NewMockedRequest("GET", "/acmplt.asmx/logo?t=1234123412341").Status(200),
		},
	})

	auth := newAuth(client)

	err := auth.sendVerifyPixel(&authenticationConfig{
		verifyPixelUrl: "acmplt.asmx/logo?t=1234123412341",
	})

	assert.Equal(t, err, nil)
}

func TestAuthenticate(t *testing.T) {
	t.Run("successful autentication", func(t *testing.T) {
		client := SetupTestClient(t, TestClientConfig{
			Mocks: []*testutils.MockedRequest{
				testutils.NewMockedRequest("GET", "/signin.aspx?bs=1").Status(200).File("testdata/auth_get_config.html"),
				testutils.NewMockedRequest("GET", "/acmplt.asmx/logo?t=1234123412341").Status(200),

				testutils.NewMockedRequest("POST", "/signin.aspx?bs=1").
					Status(200).
					File("testdata/auth_successful.html").
					MatchFormData(testutils.FormData{
						"bs":            "1",
						"cn":            "12341234134",
						"emailRestore":  "",
						"email_loc":     "",
						"oMode":         "login",
						"redirect":      "",
						"reffer":        "",
						"tmpl_filename": "signin_hvr",
						"tz":            "TestUsername", // Taken from the default HeverCredentials defined in ./auth_test.go
						"password":      "TestPassword", // Taken from the default HeverCredentials defined in ./auth_test.go
					}),
			},
		})

		auth := newAuth(client)

		err := auth.Authenticate()

		assert.Equal(t, err, nil)
		assert.Equal(t, client.isAuthenticated, true)
	})

	t.Run("unsuccessful autentication", func(t *testing.T) {
		client := SetupTestClient(t, TestClientConfig{
			Mocks: []*testutils.MockedRequest{
				testutils.NewMockedRequest("GET", "/signin.aspx?bs=1").Status(200).File("testdata/auth_get_config.html"),
				testutils.NewMockedRequest("GET", "/acmplt.asmx/logo?t=1234123412341").Status(200),
				testutils.NewMockedRequest("POST", "/signin.aspx?bs=1").Status(200).File("testdata/auth_unsuccessful.html"),
			},
		})

		auth := newAuth(client)

		err := auth.Authenticate()

		assert.ErrorIs(t, err, ErrAuthenticatedFailed)
		assert.Equal(t, client.isAuthenticated, false)
	})

}

func TestDeauthenticate(t *testing.T) {
	client := SetupTestClient(t, TestClientConfig{
		Mocks: []*testutils.MockedRequest{
			testutils.NewMockedRequest("GET", "/site/logout").Status(200),
		},
	})

	// Just for the sake of it
	client.isAuthenticated = true

	auth := newAuth(client)

	err := auth.Deauthenticate()

	assert.Equal(t, err, nil)
	assert.Equal(t, client.isAuthenticated, false)
}
