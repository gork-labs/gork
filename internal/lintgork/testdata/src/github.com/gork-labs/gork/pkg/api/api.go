package api

type Stream[E any] struct{}

type Binary struct {
	ContentType string
	Data        []byte
}

type File struct {
	Name        string
	ContentType string
	Data        []byte
}
