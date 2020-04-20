package api

type SendNotificationRequest struct {
	Message             string `json:"message"`
	DisableNotification string `json:"disable_notification"`
}

type SendNotificationResponse struct {
}

type HTTPCodedError interface {
	error
	Code() int
}

func CodedError(c int, s string) HTTPCodedError {
	return &codedError{s, c}
}

type codedError struct {
	s    string
	code int
}

func (e *codedError) Error() string {
	return e.s
}

func (e *codedError) Code() int {
	return e.code
}

var (
	MethodNotAllowedErr = CodedError(405, "Method Not Allowed")
)
