package appresp

type Response struct {
	Success bool
	Errors  []string
}

func (r *Response) AddErrorIf(isError bool, message string) {
	if !isError {
		return
	}

	r.Success = false
	r.Errors = append(r.Errors, message)
}
