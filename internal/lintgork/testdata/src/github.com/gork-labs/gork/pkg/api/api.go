package api

type Stream[E any] struct{}

type Binary struct {
	ContentType string
	Data        []byte
}
