package gohever

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

//go:generate mockery --name AuthInterface
type AuthInterface interface {
	Authenticate() error
	Deauthenticate() error
}

type Auth struct {
	hvr *Client
}

type authenticationConfig struct {
	formData       formData
	verifyPixelUrl string // Required in order to mark the session as valid
}

func newAuth(hvr *Client) *Auth {
	auth := &Auth{
		hvr: hvr,
	}

	return auth
}

func wrapAuthenticated[T any](hvr *Client, handler requestHandler[T]) requestHandler[T] {
	// Run the request handler. If we're getting any unauthenticated response, then initiate the
	// signin handler. It that fails, then... throw.

	return func() (*T, error) {
		if !hvr.isAuthenticated {
			if err := hvr.Auth.Authenticate(); err != nil {
				return nil, err
			}
		}

		result, err := handler()

		if errors.Is(err, ErrNotAuthenticated) {
			hvr.isAuthenticated = false

			if err := hvr.Auth.Authenticate(); err != nil {
				return nil, err
			}

			return handler()
		}

		return result, err
	}
}

func parseGetConfigResponse(resp *resty.Response, credentials Credentials) (*authenticationConfig, error) {
	doc, err := goquery.NewDocumentFromReader(resp.RawBody())
	if err != nil {
		return nil, err
	}

	config := &authenticationConfig{
		formData:       make(formData),
		verifyPixelUrl: "",
	}

	// Populate formData
	doc.Find("form#signinForm input[type=hidden]").Each(func(i int, s *goquery.Selection) {
		key, exists := s.Attr("name")
		val := s.AttrOr("value", "")

		if exists {
			config.formData[key] = val
		}
	})

	// Populate pixelUrl
	doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")

		if exists && strings.Index(src, "acmplt.asmx") != -1 {
			config.verifyPixelUrl = src
		}
	})

	// Update the formData with credentials
	config.formData["tz"] = credentials.Username
	config.formData["password"] = credentials.Password
	config.formData["oMode"] = "login"
	config.formData["email_loc"] = ""

	return config, nil
}

func parseAuthenticationResponse(resp *resty.Response) error {
	// There are probably better ways to check if auth was successful...
	authenticatedMatch, _ := regexp.Match("id=\"msg3\"", resp.Body())
	if resp.StatusCode() != 200 || authenticatedMatch {
		return ErrAuthenticatedFailed
	}

	return nil
}

func (auth *Auth) getConfig() (*authenticationConfig, error) {
	resp, err := auth.hvr.newRequest().
		SetDoNotParseResponse(true).
		Get(urlAuthenticate)

	if err != nil {
		return nil, err
	}

	credentials, err := auth.hvr.config.GetCredentials()
	if err != nil {
		return nil, fmt.Errorf("unable to get credentials from config: %w", err)
	}

	return parseGetConfigResponse(resp, credentials)
}

func (auth *Auth) sendVerifyPixel(config *authenticationConfig) error {
	_, err := auth.hvr.newRequest().
		Get(config.verifyPixelUrl)

	return err
}

func (auth *Auth) Authenticate() error {
	config, err := auth.getConfig()
	if err != nil {
		return err
	}

	err = auth.sendVerifyPixel(config)
	if err != nil {
		return err
	}

	resp, err := auth.hvr.newRequest().
		SetFormData(config.formData).
		Post(urlAuthenticate)

	if err != nil {
		return err
	}

	err = parseAuthenticationResponse(resp)
	if err != nil {
		return err
	}

	auth.hvr.isAuthenticated = true
	return nil
}

func (auth *Auth) Deauthenticate() error {
	_, err := auth.hvr.newRequest().
		Get(urlDeauthenticate)

	auth.hvr.isAuthenticated = false

	return err
}
