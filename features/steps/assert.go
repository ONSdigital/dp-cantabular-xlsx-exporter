package steps

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/dataset"

	"github.com/google/go-cmp/cmp"
)

func newPutVersionAssertor(b []byte) *putVersionAssertor {
	return &putVersionAssertor{
		expectedBody: b,
	}
}

// putVersionAssertor is a custom assertor function for httpfake.
// This asserts the expected request body is used when a call to the given
// endpoint is made. A custom assertor is required because godog strips the
// request body of all newlines and the default httpfake AssertBody does not
// so does return as equal even when it is correct
type putVersionAssertor struct {
	expectedBody []byte
}

func (p *putVersionAssertor) Assert(r *http.Request) error {
	var err error
	defer func() {
		err = r.Body.Close()
	}()

	var got, expected dataset.Version
	if err := json.Unmarshal(p.expectedBody, &expected); err != nil {
		return fmt.Errorf("failed to unmarshal expected body: %w", err)
	}
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	if diff := cmp.Diff(got, expected); diff != "" {
		return fmt.Errorf("request body does not match expected (-got +expected)\n%s\n", diff)
	}

	return err
}

func (p *putVersionAssertor) Log(t testing.TB) {
	t.Log("asserting request PUT version")
}

func (p *putVersionAssertor) Error(t testing.TB, err error) {
	t.Errorf("error asserting request to PUT version: %s", err)
}
