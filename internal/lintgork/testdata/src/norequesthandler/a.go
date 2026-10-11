package norequesthandler

type Request struct {
	Method string
	Path   string
	Query  string
	Status int
}
