package gohever

import (
	"crypto/tls"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/yardnsm/gohever/testutils"
)

type TestClientConfig struct {
	Authenticated bool
	Mocks []*testutils.MockedRequest
}

func SetupTestClient(t *testing.T, config TestClientConfig) *Client {
	client := NewClient(Config{
		Credentials: BasicCredentials("TestUsername", "TestPassword"),
		CreditCard: BasicCreditCard("45801234567899012", "04", "2023"),

		InitResty: func(r *resty.Client) {
			// r.SetProxy("http://127.0.0.1:8080")
			r.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
		},
	})

	server := testutils.NewMockServer().
		SetupTest(t)

	client.r.SetBaseURL(server.URL())

	for _, m := range config.Mocks {
		server.Mock(m)
	}

	if config.Authenticated {
		client.isAuthenticated = true
	}

	return client
}

func TestClientRedirectPolicy(t *testing.T) {
	t.Run("should redirect when not logged in", func(t *testing.T) {
		client := SetupTestClient(t, TestClientConfig{
			Mocks: []*testutils.MockedRequest{
				testutils.NewMockedRequest("GET", "/a").Status(302).Header("Location", "/b"),
				testutils.NewMockedRequest("GET", "/b").Status(200),
			},
		})

		client.isAuthenticated = false

		resp, err := client.newRequest().Get("a")

		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, nil, err)
	})

	t.Run("should not redirect when logged in", func(t *testing.T) {
		client := SetupTestClient(t, TestClientConfig{
			Mocks: []*testutils.MockedRequest{
				testutils.NewMockedRequest("GET", "/a").Status(302).Header("Location", "/b"),
				testutils.NewMockedRequest("GET", "/b").Status(200).ExpectNot(),
			},
		})

		client.isAuthenticated = true

		resp, err := client.newRequest().Get("a")

		assert.Equal(t, 302, resp.StatusCode())
		assert.ErrorContains(t, err, ErrRedirectIsNotAllowed.Error())
	})

	t.Run("should return an auth error if session is being logged out", func(t *testing.T) {
		client := SetupTestClient(t, TestClientConfig{
			Mocks: []*testutils.MockedRequest{
				testutils.NewMockedRequest("GET", "/loggedOut").Status(302).Header("Location", "/logout.aspx"),
			},
		})

		client.isAuthenticated = true

		resp, err := client.newRequest().Get("loggedOut")

		assert.Equal(t, 302, resp.StatusCode())
		assert.ErrorContains(t, err, ErrNotAuthenticated.Error())
	})

	t.Run("should not return an auth error when the endpoint is auth-related", func(t *testing.T) {
		client := SetupTestClient(t, TestClientConfig{
			Mocks: []*testutils.MockedRequest{
				testutils.NewMockedRequest("GET", "/site/logout").Status(302).Header("Location", "/logout.aspx"),
				testutils.NewMockedRequest("GET", "/logout.aspx").Status(200),
			},
		})

		client.isAuthenticated = true

		resp, err := client.newRequest().Get("site/logout")

		assert.Equal(t, 200, resp.StatusCode())
		assert.NoError(t, err)
	})
}
