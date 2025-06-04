package apperrors

//go:generate go tool github.com/mailru/easyjson/easyjson -snake_case -all -no_std_marshalers ./json.go

type JSONError struct {
	Success bool // always false
	Message string
}

func (e *JSONError) Error() string {
	return e.Message
}
