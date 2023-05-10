package gohever

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

var (
	regexCardFactor1 = regexp.MustCompile("var gift_card_factor1 = ([0-9\\.]+);")
	regexCardFactor2 = regexp.MustCompile("var gift_card_factor2 = ([0-9\\.]+);")
	regexCardFactor3 = regexp.MustCompile("var gift_card_factor3 = ([0-9\\.]+);")

	regexCardFactorPrice1 = regexp.MustCompile("var gift_card_factor1_price = ([0-9\\.]+);")
	regexCardFactorPrice2 = regexp.MustCompile("var gift_card_factor2_price = ([0-9\\.]+);")
	regexCardFactorPrice3 = regexp.MustCompile("var gift_card_factor3_price = ([0-9\\.]+);")

	regexMaxMonthLoad = regexp.MustCompile("var max_month_load = ([0-9\\.]+);")
	regexMaxOnCard    = regexp.MustCompile("var max_on_card = ([0-9\\.]+);")

	regexSerialNumber = regexp.MustCompile("name=\"sn\" value=\"((?:\\w|-)+)\"")

	regexLoadStatusCode = regexp.MustCompile("if \\( (\\d) == 1 \\)")
	regexPlainNumber    = regexp.MustCompile("\\d+")
)

type CardInterface interface {
	Type() CardType

	GetStatus() (*CardStatus, error)
	GetHistory() (*[]CardHistoryItem, error)
	Load(status CardStatus, amount int32) (*LoadResult, error)
}

type Card struct {
	hvr      *Client
	cardType CardType
}

// Represents a factor in the card, for example 30% for 1000ILS
type CardFactor struct {
	Factor float64
	Amount float64
}

// The status of a card at a given time
type CardStatus struct {
	// The factors "steps" in the card, ordered accordingly. The "Amount" property means the maximum
	// amount the factor can take.
	Factors []CardFactor

	// The maximum amount we can load the card monthly
	MaxMonthlyAmount int

	// The maximum amount that the card can hold at a given time
	MaxOnCardAmount int

	// The current load on the card
	CurrentBalance float64

	// The remaining load until the end of the month
	RemainingMonthlyAmount float64

	// The remaning load until the card will be full
	RemainingOnCardAmount float64

	// The total monthly usage
	MonthlyUsage float64

	// The balance left from previous month and does not count against the current mothly quota
	Leftovers float64

	// Serial number (internal, used for charging the card)
	SerialNumber string
}

// The result of a load estimation
type CardEstimate struct {
	// The final estimation
	Total         float64
	TotalFactored float64

	// The amount needed to load in order to reach the desired estimation
	Required         float64
	RequiredFactored float64

	// Amount taken from leftovers, does not include the factors
	Leftovers float64

	// Amount taken from factors. The "Amount" property means the amount taken from the factor
	// in order to reach the total.
	Factors []CardFactor
}

type CardHistoryItem struct {
	Id           string
	Date         string
	ActionType   CardAction
	BusinessName string
	Amount       float64
}

// Card types
type CardType int

const (
	TypeKeva CardType = iota
	TypeTeamim
)

// Card history actions
type CardAction int

const (
	ActionLoad CardAction = iota
	ActionPurchase
)

// The status of a card load
type LoadStatus int

const (
	StatusNone LoadStatus = iota
	StatusError
	StatusSuccess
)

type LoadResult struct {
	Status     LoadStatus
	LoadNumber string
	RawMessage string
}

// The card config parsed from the site, used internally in this package
type cardConfig struct {
	cardFactor1      float64
	cardFactor2      float64
	cardFactor3      float64
	cardFactorPrice1 float64
	cardFactorPrice2 float64
	cardFactorPrice3 float64
	maxMonthLoad     int
	maxOnCard        int
	serialNumber     string
}

// The card balance parsed from the site, used internally in this package
type cardBalance struct {
	currentBalance         float64
	remainingMonthlyAmount float64
	remainingOnCardAmount  float64
}

func newCard(hvr *Client, cardType CardType) *Card {
	card := &Card{
		hvr:      hvr,
		cardType: cardType,
	}

	return card
}

func parseGetCardConfigResponse(resp *resty.Response) (*cardConfig, error) {
	body := string(resp.Body())

	var (
		cardFactor1 float64
		cardFactor2 float64
		cardFactor3 float64

		cardFactorPrice1 float64
		cardFactorPrice2 float64
		cardFactorPrice3 float64

		maxMonthLoad float64
		maxOnCard    float64

		serialNumber string
	)

	scanMap := map[*float64]*regexp.Regexp{
		&cardFactor1: regexCardFactor1,
		&cardFactor2: regexCardFactor2,
		&cardFactor3: regexCardFactor3,

		&cardFactorPrice1: regexCardFactorPrice1,
		&cardFactorPrice2: regexCardFactorPrice2,
		&cardFactorPrice3: regexCardFactorPrice3,

		&maxMonthLoad: regexMaxMonthLoad,
		&maxOnCard:    regexMaxOnCard,
	}

	for ptr, reg := range scanMap {
		val, err := strconv.ParseFloat(reg.FindStringSubmatch(body)[1], 64)
		if err != nil {
			return nil, err
		}

		*ptr = val
	}

	serialNumber = regexSerialNumber.FindStringSubmatch(body)[1]

	return &cardConfig{
		cardFactor1,
		cardFactor2,
		cardFactor3,

		cardFactorPrice1,
		cardFactorPrice2,
		cardFactorPrice3,

		int(maxMonthLoad),
		int(maxOnCard),

		serialNumber,
	}, nil
}

func parseGetCardBalanceResponse(resp *resty.Response) (*cardBalance, error) {
	body := string(resp.Body())
	parts := strings.Split(body, "|")

	var (
		currentBalance         float64
		remainingMonthlyAmount float64
		remainingOnCardAmount  float64
	)

	scanMap := map[*float64]int{
		&currentBalance:         0,
		&remainingMonthlyAmount: 1,
		&remainingOnCardAmount:  2,
	}

	for ptr, partIndex := range scanMap {
		part := strings.ReplaceAll(strings.TrimSpace(parts[partIndex]), ",", "")
		val, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return nil, err
		}

		*ptr = val
	}

	return &cardBalance{
		currentBalance,
		remainingMonthlyAmount,
		remainingOnCardAmount,
	}, nil
}

func parseGetCardHistoryResponse(resp *resty.Response) (*[]CardHistoryItem, error) {
	doc, err := goquery.NewDocumentFromReader(resp.RawBody())
	if err != nil {
		return nil, err
	}

	var history []CardHistoryItem

	// Populate formData
	doc.Find("tr.historyRows[id]").Each(func(i int, s *goquery.Selection) {
		var item CardHistoryItem

		item.Id = s.AttrOr("id", "no_id_"+strconv.Itoa(i))
		item.Date = s.Find("td:nth-child(1)").Text()

		item.ActionType = ActionLoad

		if s.Find("td:nth-child(2)").Text() == "רכישה" {
			item.ActionType = ActionPurchase
		}

		item.BusinessName = s.Find("td:nth-child(3)").Text()

		amount, err := strconv.ParseFloat(s.Find("td:nth-child(4)").Text(), 64)
		if err == nil {
			item.Amount = amount
		}

		history = append(history, item)
	})

	return &history, nil
}

func parseLoadCardResponse(resp *resty.Response) (*LoadResult, error) {
	body := string(resp.Body())

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	var (
		status     LoadStatus
		loadNumber string
		rawMessage string
	)

	// A very naive check for error.
	// A better check will be to search for the error table, such in ./testdata/card_load_error.html
	if !regexLoadStatusCode.MatchString(body) {
		return &LoadResult{
			Status:     StatusError,
			LoadNumber: "",
			RawMessage: strings.TrimSpace(doc.Find("table.table[bgcolor=red]").Text()),
		}, nil
	}

	// First we'll get the status code of the request. Yeah, this is also from the javasript they're
	// using within...
	statucCode, err := strconv.ParseInt(regexLoadStatusCode.FindStringSubmatch(body)[1], 10, 64)
	switch statucCode {
	case 0:
		status = StatusNone
	case 1:
		status = StatusError
	case 2:
		status = StatusSuccess
	}

	rawMessage = strings.TrimSpace(doc.Find("div#msg_ok").Text())
	loadNumber = regexPlainNumber.FindString(rawMessage)

	return &LoadResult{
		Status:     status,
		LoadNumber: loadNumber,
		RawMessage: rawMessage,
	}, nil
}

func (card *Card) buildBaseRequest() *resty.Request {
	req := card.hvr.newRequest()

	if card.cardType == TypeTeamim {
		req.SetQueryParam(queryParamFoodCard, "1")
	}

	return req
}

func (card *Card) getCardConfig() (*cardConfig, error) {
	resp, err := card.buildBaseRequest().Get(urlCardConfig)

	if err != nil {
		return nil, err
	}

	return parseGetCardConfigResponse(resp)
}

func (card *Card) getCardBalance(config *cardConfig) (*cardBalance, error) {
	resp, err := card.buildBaseRequest().
		SetFormData(formData{
			"balance_only":           "1",
			"current_max_month_load": strconv.Itoa(config.maxMonthLoad),
			"current_max_load":       strconv.Itoa(config.maxOnCard),
		}).
		Post(urlCardStatus)

	if err != nil {
		return nil, err
	}

	return parseGetCardBalanceResponse(resp)
}

func (card *Card) getCardHistory() (*[]CardHistoryItem, error) {
	resp, err := card.buildBaseRequest().
		SetDoNotParseResponse(true).
		Get(urlCardHistory)

	if err != nil {
		return nil, err
	}

	return parseGetCardHistoryResponse(resp)
}

func (card *Card) loadCard(status CardStatus, amount int32) (*LoadResult, error) {
	creditCard, err := card.hvr.config.GetCreditCard()
	if err != nil {
		return nil, fmt.Errorf("unable to get credit card details from config: %w", err)
	}

	resp, err := card.buildBaseRequest().
		SetFormData(formData{
			"price":      strconv.Itoa(int(amount)),
			"card_num":   creditCard.Number,
			"card_year":  creditCard.Year,
			"card_month": creditCard.Month,

			"chkTakanon": "",
			"om":         "load",
			"req_sent":   "1",

			"sn": status.SerialNumber,
		}).
		Post(urlLoadCard)

	if err != nil {
		return nil, err
	}

	return parseLoadCardResponse(resp)
}

func (card *Card) Type() CardType {
	return card.cardType
}

func (card *Card) GetStatus() (*CardStatus, error) {
	return wrapAuthenticated(card.hvr, func() (*CardStatus, error) {
		config, err := card.getCardConfig()
		if err != nil {
			return nil, err
		}

		balance, err := card.getCardBalance(config)
		if err != nil {
			return nil, err
		}

		monthlyUsage := 1 - (balance.remainingMonthlyAmount / float64(config.maxMonthLoad))
		leftovers := math.Max(0,
			balance.currentBalance-monthlyUsage*balance.remainingMonthlyAmount)

		return &CardStatus{
			Factors: []CardFactor{
				{Factor: config.cardFactor1, Amount: config.cardFactorPrice1},
				{Factor: config.cardFactor2, Amount: config.cardFactorPrice2},
				{Factor: config.cardFactor3, Amount: config.cardFactorPrice3},
			},

			MaxMonthlyAmount: config.maxMonthLoad,
			MaxOnCardAmount:  config.maxOnCard,

			CurrentBalance:         balance.currentBalance,
			RemainingMonthlyAmount: balance.remainingMonthlyAmount,
			RemainingOnCardAmount:  balance.remainingOnCardAmount,

			MonthlyUsage: monthlyUsage,
			Leftovers:    leftovers,

			SerialNumber: config.serialNumber,
		}, nil
	})()
}

func (card *Card) GetHistory() (*[]CardHistoryItem, error) {
	return wrapAuthenticated(card.hvr, func() (*[]CardHistoryItem, error) {
		return card.getCardHistory()
	})()
}

func (card *Card) Load(status CardStatus, amount int32) (*LoadResult, error) {
	return wrapAuthenticated(card.hvr, func() (*LoadResult, error) {
		return card.loadCard(status, amount)
	})()
}

func (status *CardStatus) Estimate(amount float64) (*CardEstimate, error) {
	if amount < 0 {
		return nil, ErrLoadInvalidValue
	}

	if amount < minimumLoadAmount {
		return nil, ErrNotEnoughToLoad
	}

	if amount > status.RemainingMonthlyAmount {
		return nil, ErrLoadAboveMonthlyLimit
	}

	if amount+status.CurrentBalance > float64(status.MaxOnCardAmount) {
		return nil, ErrLoadAboveOnCardLimit
	}

	var (
		total         float64
		totalFactored float64

		required         float64
		requiredFactored float64

		leftovers float64
		factors   []CardFactor
	)

	// Take from leftovers
	leftovers = math.Min(
		status.Leftovers,
		amount,
	)

	total += leftovers
	totalFactored += leftovers * status.Factors[len(status.Factors)-1].Factor

	// The used monthly used balance (leftovers and current balance aside)
	usedBalance := float64(status.MaxMonthlyAmount) -
		status.RemainingMonthlyAmount -
		status.CurrentBalance

	// The accumulated factors sum
	factorsSum := 0.0

	for _, factor := range status.Factors {
		// The remaining amount from this factor level
		factorsSum += factor.Amount
		remaining := math.Max(0, factorsSum-usedBalance)

		// Take from remaining
		taken := math.Min(
			remaining,
			math.Max(0, amount-total),
		)

		usedBalance += taken

		// This defines the diff between the taken amount (so far) and the current balance on the
		// card
		factorRequired := math.Max(0, taken-math.Max(
			0,
			status.CurrentBalance-total,
		))

		total += taken
		totalFactored += taken * factor.Factor

		required += factorRequired
		requiredFactored += factorRequired * factor.Factor

		factors = append(factors, CardFactor{
			Amount: taken,
			Factor: factor.Factor,
		})
	}

	return &CardEstimate{
		Total:         total,
		TotalFactored: totalFactored,

		Required:         required,
		RequiredFactored: requiredFactored,

		Leftovers: leftovers,
		Factors:   factors,
	}, nil
}
