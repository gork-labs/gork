package api

type Stream[E any] struct{}

type File struct {
	Name        string
	ContentType string
	Data        []byte
}
