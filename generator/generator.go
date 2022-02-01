package generator

import (
	"crypto/rand"
	"fmt"
)

// Generator is responsible for randomly generating new strings and tokens
// that might need to be mocked out to produce consistent output for tests
type Generator struct{}

// New returns a new Generator
func New() *Generator {
	return &Generator{}
}

// NewPSK returns a new random array of 16 bytes
func (g *Generator) NewPSK() ([]byte, error) {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to read random bytes into key: %w", err)
	}

	return key, nil
}
