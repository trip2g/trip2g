package appreq

// Error represents an error with a status code and message.
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// SystemMessageError asks the router to answer with a page a person can read
// instead of the failure itself. Msg names the situation; the router resolves
// it to localized copy. Err is the technical cause — it goes to the log, never
// into the response.
type SystemMessageError struct {
	Code int
	Msg  string
	Err  error
}

func (e *SystemMessageError) Error() string {
	if e.Err == nil {
		return e.Msg
	}

	return e.Msg + ": " + e.Err.Error()
}

func (e *SystemMessageError) Unwrap() error {
	return e.Err
}
