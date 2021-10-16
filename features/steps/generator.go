package steps

import (
	"crypto/rand"
)

// Generator is responsible for randomly generating new strings and tokens
// that might need to be mocked out to produce consistent output for tests
type generator struct{}

// NewPSK returns a new random array of 16 bytes
func (g *generator) NewPSK() ([]byte, error) {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	return key, nil
}
