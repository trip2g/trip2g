package appreq

// Error represents an error with a status code and message.
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
