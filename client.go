package gohever

import (
	"net/http"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	config Config
	r      *resty.Client

	isAuthenticated bool

	Auth  AuthInterface
	Cards struct {
		Keva   CardInterface
		Teamim CardInterface
	}
}

func NewClient(config Config) *Client {
	r := resty.New()

	client := &Client{
		config: config,
		r:      r,

		isAuthenticated: false,
	}

	client.init()

	return client
}

func (hvr *Client) init() {

	// Setup endpoints
	hvr.Auth = newAuth(hvr)
	hvr.Cards.Keva = newCard(hvr, TypeKeva)
	hvr.Cards.Teamim = newCard(hvr, TypeTeamim)

	// Setup resty
	hvr.r.SetBaseURL(heverBaseUrl)
	hvr.r.SetRedirectPolicy(
		resty.RedirectPolicyFunc(hvr.redirectPolicy),
	)

	// Common headers
	hvr.r.SetHeader("User-Agent", heverUserAgent)
	hvr.r.SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")

	if hvr.config.InitResty != nil {
		hvr.config.InitResty(hvr.r)
	}
}

func (hvr *Client) newRequest() *resty.Request {
	return hvr.r.NewRequest()
}

func (hvr *Client) redirectPolicy(req *http.Request, via []*http.Request) error {
	// We'll be abusing redirects to check whether the user is authenticated after a request.
	// However, redirecting is needed when authenticating because of a shitty chain they got in the
	// flow, so we're going to allow redirecting only when the user is *not* authenticated.

	// Do not catch authentication requests
	requester := via[0].URL.Path[1:]
	if requester == urlAuthenticate || requester == urlDeauthenticate {
		return nil
	}

	// should be the same as ErrNotAuthenticated
	if req.URL.Path == "/logout.aspx" || (hvr.isAuthenticated && req.URL.Path[1:] == urlDeauthenticate) {
		hvr.isAuthenticated = false
		return ErrNotAuthenticated
	}

	if hvr.isAuthenticated {
		return ErrRedirectIsNotAllowed
	}

	return nil
}
