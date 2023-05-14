# gohever

> 💳 An API for the Hever website, written in Go.

[![GitHub Actions](https://github.com/yardnsm/gohever/actions/workflows/go.yml/badge.svg)](https://github.com/yardnsm/gohever/actions)

```bash
go get -u github.com/yardnsm/gohever
```

* [Overview](#overview)
* [License](#license)

## Overview

gohever is an API wrapper around Hever's website. I originally wanted to build a Telegram bot that
allows me to load my cards without logging in to the website and waiting for the ENORMOUS assets to
load, and so "gohever" was born.

This package provides the basic functionality that the website offers and some nice features, such
as:

* Card information retrieval: balance, monthly usage, leftovers from the previous month, information
  regarding the discounts, etc.
* Card history
* Load estimation using the retrieved card status.
* Loading the card using your HEVER credit card.

Plus some nice things that I really like:

* Nice [testutils](./testutils) for making testing the client way easier;
* Automatic handling of authentication - you don't need to call `Authenticate()` at all!

⚠️ **This project was meant to be used for educational purposes only. I am not affiliated with Hever in any way.**

#### A note on `testdata`

The thing is - some data that this package parse comes from the HTML or JavaScript response of the
website. This content is not suitable for open-source (and if it is, I don't want to find out),
so I keep it in a separate, private repo.

---

## License

MIT © [Yarden Sod-Moriah](https://ysm.sh/)
