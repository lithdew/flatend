package flatend

type Error struct {
	Status int
	Err    error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}
