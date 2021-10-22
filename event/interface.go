package event

import (
	"context"
)

//go:generate moq -out mock/handler.go -pkg mock . Handler

type Handler interface {
	Handle(ctx context.Context, csvComplete *CantabularCsvCreated) error
}

type dataLogger interface { //!!! figure out where and how this is used
	LogData() map[string]interface{}
}

type coder interface {
	Code() int
}
