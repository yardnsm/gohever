package gohever

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yardnsm/gohever/testutils"
)

// Create a classic card mimicing an authentic HEVER card
func setupCardStatus(prevMonthlyUsage, onCard, leftovers float64) CardStatus {
	return CardStatus{
		Factors: []CardFactor{
			{Factor: 0.7, Amount: 1000},
			{Factor: 0.75, Amount: 1000},
			{Factor: 0.8, Amount: 1000},
		},

		MaxMonthlyAmount: 3000,
		MaxOnCardAmount:  1000,

		CurrentBalance:         onCard,
		RemainingMonthlyAmount: 3000 - prevMonthlyUsage - onCard + leftovers,
		RemainingOnCardAmount:  1000 - onCard,

		MonthlyUsage: 0, // Irrelevant for estimations
		Leftovers:    leftovers,

		SerialNumber: "12345678-9abc-def1-2345-6789abcdef12",
	}
}

func TestCardBaseConfig(t *testing.T) {
	t.Run("keva card", func(t *testing.T) {
		client := SetupTestClient(t, TestClientConfig{
			Mocks: []*testutils.MockedRequest{
				testutils.NewMockedRequest("GET", "/some_path").Once().Status(200),
			},
		})

		card := newCard(client, TypeKeva)
		card.buildBaseRequest().Get("some_path")
	})

	t.Run("teamim card", func(t *testing.T) {
		client := SetupTestClient(t, TestClientConfig{
			Mocks: []*testutils.MockedRequest{
				testutils.NewMockedRequest("GET", "/some_path?food=1").Once().Status(200),
			},
		})

		card := newCard(client, TypeTeamim)
		card.buildBaseRequest().Get("some_path")
	})
}

func TestGetCardConfig(t *testing.T) {
	client := SetupTestClient(t, TestClientConfig{
		Mocks: []*testutils.MockedRequest{
			testutils.NewMockedRequest("GET", "/orders/gift_2000.aspx").Status(200).File("testdata/card_get_config.html"),
		},
	})

	card := newCard(client, TypeKeva)

	config, _ := card.getCardConfig()

	assert.Equal(t, &cardConfig{
		cardFactor1: 0.9876,
		cardFactor2: 0.5432,
		cardFactor3: 0.80,

		cardFactorPrice1: 1029,
		cardFactorPrice2: 2837,
		cardFactorPrice3: 1000,

		maxMonthLoad: 4321,
		maxOnCard:    1234,

		serialNumber: "12345678-9abc-def1-2345-6789abcdef12",
	}, config)
}

func TestCardGetBalance(t *testing.T) {
	client := SetupTestClient(t, TestClientConfig{
		Mocks: []*testutils.MockedRequest{
			testutils.NewMockedRequest("POST", "/orders/gift_2000.aspx").
				Status(200).
				File("testdata/card_get_balance.html").
				MatchFormData(testutils.FormData{
					"balance_only":           "1",
					"current_max_month_load": "4321",
					"current_max_load":       "1234",
				}),
		},
	})

	card := newCard(client, TypeKeva)

	balance, err := card.getCardBalance(&cardConfig{
		cardFactor1:      0.9876,
		cardFactor2:      0.5432,
		cardFactor3:      0.80,
		cardFactorPrice1: 1029,
		cardFactorPrice2: 2837,
		cardFactorPrice3: 1000,
		maxMonthLoad:     4321,
		maxOnCard:        1234,
	})

	t.Logf("%q", err)

	assert.Equal(t, &cardBalance{
		currentBalance:         512,
		remainingMonthlyAmount: 3809,
		remainingOnCardAmount:  722,
	}, balance)
}

func TestCardGetStatus(t *testing.T) {
	client := SetupTestClient(t, TestClientConfig{
		Authenticated: true,
		Mocks: []*testutils.MockedRequest{
			testutils.NewMockedRequest("GET", "/orders/gift_2000.aspx").Status(200).File("testdata/card_get_config.html"),
			testutils.NewMockedRequest("POST", "/orders/gift_2000.aspx").Status(200).File("testdata/card_get_balance.html"),
		},
	})

	card := newCard(client, TypeKeva)

	status, _ := card.GetStatus()

	assert.Equal(t, &CardStatus{
		Factors: []CardFactor{
			{Factor: 0.9876, Amount: 1029},
			{Factor: 0.5432, Amount: 2837},
			{Factor: 0.80, Amount: 1000},
		},

		MaxMonthlyAmount: 4321,
		MaxOnCardAmount:  1234,

		CurrentBalance:         512,
		RemainingMonthlyAmount: 3809,
		RemainingOnCardAmount:  722,

		MonthlyUsage: 0.1184910900254571,
		Leftovers:    60.6674380930339,

		SerialNumber: "12345678-9abc-def1-2345-6789abcdef12",
	}, status)
}

func TestCardGetHistory(t *testing.T) {
	client := SetupTestClient(t, TestClientConfig{
		Authenticated: true,
		Mocks: []*testutils.MockedRequest{
			testutils.NewMockedRequest("GET", "/orders/gift_2000.aspx").Status(200).File("testdata/card_get_history.html"),
		},
	})

	card := newCard(client, TypeKeva)

	history, _ := card.GetHistory()

	assert.Equal(t, &[]CardHistoryItem{
		{Id: "year_2022_1", Date: "22/01/2022", ActionType: ActionPurchase, BusinessName: "Business X", Amount: -794.6},
		{Id: "year_2022_2", Date: "24/01/2022", ActionType: ActionPurchase, BusinessName: "Business Y", Amount: -204},
		{Id: "year_2022_3", Date: "16/02/2022", ActionType: ActionLoad, BusinessName: "-", Amount: 144},
		{Id: "year_2022_4", Date: "16/02/2022", ActionType: ActionLoad, BusinessName: "-", Amount: 5},
		{Id: "year_2022_5", Date: "16/02/2022", ActionType: ActionPurchase, BusinessName: "Business Z", Amount: -144.9},
	}, history)
}

func TestCardStatusEstimate(t *testing.T) {
	t.Run("taking all from leftovers", func(t *testing.T) {
		cardStatus := setupCardStatus(0, 450, 450)
		estimate, _ := cardStatus.Estimate(100)

		assert.Equal(t, &CardEstimate{
			Total:         100,
			TotalFactored: 80, // 100*0.8, leftovers considered as the last factor

			Required:         0,
			RequiredFactored: 0,

			Leftovers: 100,
			Factors: []CardFactor{
				{Factor: 0.7, Amount: 0},
				{Factor: 0.75, Amount: 0},
				{Factor: 0.8, Amount: 0},
			},
		}, estimate)
	})

	t.Run("taking more from leftovers and some from factors", func(t *testing.T) {
		cardStatus := setupCardStatus(0, 450, 450)
		estimate, _ := cardStatus.Estimate(550)

		assert.Equal(t, &CardEstimate{
			Total:         550,
			TotalFactored: 430, // 450*0.8 + 100 * 0.7

			Required:         100,
			RequiredFactored: 70, // 100*0.7

			Leftovers: 450,
			Factors: []CardFactor{
				{Factor: 0.7, Amount: 100},
				{Factor: 0.75, Amount: 0},
				{Factor: 0.8, Amount: 0},
			},
		}, estimate)
	})

	t.Run("taking from first factor only", func(t *testing.T) {
		cardStatus := setupCardStatus(300, 200, 0)
		estimate, _ := cardStatus.Estimate(450)

		assert.Equal(t, &CardEstimate{
			Total:         450,
			TotalFactored: 315, // 450*0.7

			Required:         250,
			RequiredFactored: 175, // 250*0.7

			Leftovers: 0,
			Factors: []CardFactor{
				{Factor: 0.7, Amount: 450},
				{Factor: 0.75, Amount: 0},
				{Factor: 0.8, Amount: 0},
			},
		}, estimate)
	})

	t.Run("taking from two factors", func(t *testing.T) {
		cardStatus := setupCardStatus(
			700, // We're almost passing the first factor
			200, // Total of 900 monthly usage. 100 more to end the first factor.
			0,
		)

		estimate, _ := cardStatus.Estimate(700)

		assert.Equal(t, &CardEstimate{
			Total:         700,
			TotalFactored: 510, // 300*0.7 + 400*0.75

			Required:         500,
			RequiredFactored: 370, // 100*0.7 +  400* 0.75

			Leftovers: 0,
			Factors: []CardFactor{
				{Factor: 0.7, Amount: 300},
				{Factor: 0.75, Amount: 400},
				{Factor: 0.8, Amount: 0},
			},
		}, estimate)
	})

	t.Run("taking from the last factor", func(t *testing.T) {
		cardStatus := setupCardStatus(2100, 200, 0)
		estimate, _ := cardStatus.Estimate(700)

		assert.Equal(t, &CardEstimate{
			Total:         700,
			TotalFactored: 560, // 700*0.8

			Required:         500,
			RequiredFactored: 400, // 500*0.8

			Leftovers: 0,
			Factors: []CardFactor{
				{Factor: 0.7, Amount: 0},
				{Factor: 0.75, Amount: 0},
				{Factor: 0.8, Amount: 700},
			},
		}, estimate)
	})

	t.Run("passing maximum on card amount", func(t *testing.T) {
		cardStatus := setupCardStatus(300, 200, 0)
		_, err := cardStatus.Estimate(900)

		assert.ErrorIs(t, err, ErrLoadAboveOnCardLimit)
	})

	t.Run("passing monthly amount", func(t *testing.T) {
		cardStatus := setupCardStatus(2700, 200, 0)
		_, err := cardStatus.Estimate(400)

		assert.ErrorIs(t, err, ErrLoadAboveMonthlyLimit)
	})

	t.Run("passing invalid value", func(t *testing.T) {
		cardStatus := setupCardStatus(2700, 200, 0)
		_, err := cardStatus.Estimate(-400)

		assert.ErrorIs(t, err, ErrLoadInvalidValue)
	})

	t.Run("passing less than the minmum amount", func(t *testing.T) {
		cardStatus := setupCardStatus(2700, 200, 0)
		_, err := cardStatus.Estimate(3)

		assert.ErrorIs(t, err, ErrNotEnoughToLoad)
	})
}

func TestCardLoad(t *testing.T) {
	t.Run("successful load", func(t *testing.T) {
		client := SetupTestClient(t, TestClientConfig{
			Authenticated: true,
			Mocks: []*testutils.MockedRequest{
				testutils.NewMockedRequest("POST", "/orders/gift_2000.aspx").
					Status(200).
					File("testdata/card_load_success.html").
					MatchFormData(testutils.FormData{
						"price":      "500",
						"card_num":   "45801234567899012", // Taken from the default HeverCreditCard defined in ./auth_test.go
						"card_year":  "2023",              // Taken from the default HeverCreditCard defined in ./auth_test.go
						"card_month": "04",                // Taken from the default HeverCreditCard defined in ./auth_test.go

						"chkTakanon": "",
						"om":         "load",
						"req_sent":   "1",

						"sn": "12345678-9abc-def1-2345-6789abcdef12",
					}),
			},
		})

		card := newCard(client, TypeKeva)
		status := setupCardStatus(400, 0, 0)

		result, err := card.Load(status, 500)

		assert.Equal(t, nil, err)
		assert.Equal(t, &LoadResult{
			Status:     StatusSuccess,
			LoadNumber: "12344321",
			RawMessage: "בקשת טעינת הכרטיס בוצעה. מספר ההזמנה: 12344321",
		}, result)
	})

	t.Run("failure load", func(t *testing.T) {
		client := SetupTestClient(t, TestClientConfig{
			Authenticated: true,
			Mocks: []*testutils.MockedRequest{
				testutils.NewMockedRequest("POST", "/orders/gift_2000.aspx").Status(200).File("testdata/card_load_error.html"),
			},
		})

		card := newCard(client, TypeKeva)
		status := setupCardStatus(400, 0, 0)

		result, err := card.Load(status, 500)

		assert.Equal(t, nil, err)
		assert.Equal(t, &LoadResult{
			Status:     StatusError,
			LoadNumber: "",
			RawMessage: "הפעילויות המוצעות באתר מיועדות לעמיתי \"חבר\" בלבד, בהתאם לנתוניו האישיים של העמית וסל הזכאות שנקבע.עפ\"י רישומינו, כרטיס האשראי שהקשת אינו מעודכן ככרטיס אשראי המשויך ל\"חבר\" או שאינו תואם לתעודת הזהות איתה התבצעה הכניסה לאתר.אנא ודא כי תעודת הזהות ופרטי כרטיס האשראי שהקשת נכונים ותואמים זה לזה.במידה והפרטים נכונים – אנא פנה אלינו באמצעות קישור \"צור קשר\" בדף הבית באתר.בפנייתך אנא ציין שלא הצלחת לבצע הזמנה ואת הפרטים הבאים: מספר אישי, מספר תעודת זהות, ו- 4 ספרות אחרונות של כרטיס האשראי עימו הינך מבקש לבצע את הפעולה.!",
		}, result)
	})
}
