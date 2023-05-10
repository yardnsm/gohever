package gohever

// The reason handler will return a pointer rather than a plain value is that we want the handler to
// allow returning nil values as results

//go:generate mockery --name requestHandler --exported
type requestHandler[T any] func() (*T, error)

type formData map[string]string
