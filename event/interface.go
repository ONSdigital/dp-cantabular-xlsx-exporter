package event

import (
	"context"
)

//go:generate moq -out mock/handler.go -pkg mock . Handler

type Handler interface {
	Handle(ctx context.Context, XlsxCreate *CantabularCsvCreated) error
}

type dataLogger interface {
	LogData() map[string]interface{}
}

type coder interface {
	Code() int
}
