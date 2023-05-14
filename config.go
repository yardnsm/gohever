package gohever

import "github.com/go-resty/resty/v2"

type Credentials struct {
	Username string
	Password string
}

type CreditCard struct {
	Number string
	Month  string
	Year   string
}

type Config struct {
	InitResty      func(r *resty.Client)
	Credentials func() (Credentials, error)
	CreditCard  func() (CreditCard, error)
}

func BasicCredentials(username, password string) func() (Credentials, error) {
	return func() (Credentials, error) {
		return Credentials{
			Username: username,
			Password: password,
		}, nil
	}
}

func BasicCreditCard(number, month, year string) func() (CreditCard, error) {
	return func() (CreditCard, error) {
		return CreditCard{
			Number: number,
			Month: month,
			Year: year,
		}, nil
	}
}
