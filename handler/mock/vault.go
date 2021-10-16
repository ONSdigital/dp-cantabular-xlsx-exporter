// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mock

import (
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler"
	"sync"
)

var (
	lockVaultClientMockWriteKey sync.RWMutex
)

// Ensure, that VaultClientMock does implement handler.VaultClient.
// If this is not the case, regenerate this file with moq.
var _ handler.VaultClient = &VaultClientMock{}

// VaultClientMock is a mock implementation of handler.VaultClient.
//
//     func TestSomethingThatUsesVaultClient(t *testing.T) {
//
//         // make and configure a mocked handler.VaultClient
//         mockedVaultClient := &VaultClientMock{
//             WriteKeyFunc: func(path string, key string, value string) error {
// 	               panic("mock out the WriteKey method")
//             },
//         }
//
//         // use mockedVaultClient in code that requires handler.VaultClient
//         // and then make assertions.
//
//     }
type VaultClientMock struct {
	// WriteKeyFunc mocks the WriteKey method.
	WriteKeyFunc func(path string, key string, value string) error

	// calls tracks calls to the methods.
	calls struct {
		// WriteKey holds details about calls to the WriteKey method.
		WriteKey []struct {
			// Path is the path argument value.
			Path string
			// Key is the key argument value.
			Key string
			// Value is the value argument value.
			Value string
		}
	}
}

// WriteKey calls WriteKeyFunc.
func (mock *VaultClientMock) WriteKey(path string, key string, value string) error {
	if mock.WriteKeyFunc == nil {
		panic("VaultClientMock.WriteKeyFunc: method is nil but VaultClient.WriteKey was just called")
	}
	callInfo := struct {
		Path  string
		Key   string
		Value string
	}{
		Path:  path,
		Key:   key,
		Value: value,
	}
	lockVaultClientMockWriteKey.Lock()
	mock.calls.WriteKey = append(mock.calls.WriteKey, callInfo)
	lockVaultClientMockWriteKey.Unlock()
	return mock.WriteKeyFunc(path, key, value)
}

// WriteKeyCalls gets all the calls that were made to WriteKey.
// Check the length with:
//     len(mockedVaultClient.WriteKeyCalls())
func (mock *VaultClientMock) WriteKeyCalls() []struct {
	Path  string
	Key   string
	Value string
} {
	var calls []struct {
		Path  string
		Key   string
		Value string
	}
	lockVaultClientMockWriteKey.RLock()
	calls = mock.calls.WriteKey
	lockVaultClientMockWriteKey.RUnlock()
	return calls
}
