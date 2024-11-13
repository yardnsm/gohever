package gohever

import "errors"

const (
	mccBaseUrl   = "https://www.mcc.co.il/"
	heverBaseUrl = "https://www.hvr.co.il/"

	heverUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36"
)

// Endpoints
const (
	urlAuthenticate   = "signin.aspx?bs=1"
	urlDeauthenticate = "site/logout"

	urlCardConfig  = "orders/gift_2000.aspx"
	urlCardStatus  = "orders/gift_2000.aspx"
	urlCardHistory = "orders/gift_2000.aspx"
	urlLoadCard    = "orders/gift_2000.aspx"
)

// Query Params
const (
	queryParamFoodCard = "food"
)

// Parameters
const (
	minimumLoadAmount = 5
)

// Errors
var (
	ErrRedirectIsNotAllowed = errors.New("request redirect is not allowed")

	ErrNotAuthenticated    = errors.New("not authenticated to HEVER website")
	ErrAuthenticatedFailed = errors.New("failed to authenticate to HEVER website")

	ErrUnableToParseCardConfig = errors.New("failed to parse the card config")

	ErrNotEnoughToLoad       = errors.New("the amount to load should be above 5")
	ErrLoadAboveOnCardLimit  = errors.New("charging above the max on card limit")
	ErrLoadAboveMonthlyLimit = errors.New("charging above the max monthly limit")
	ErrLoadInvalidValue      = errors.New("invalid value was passed to load")
)
