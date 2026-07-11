package container

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Driver struct {
	baseURL string
	client  *http.Client
}

func New(baseURL string) *Driver {
	return &Driver{baseURL: baseURL, client: http.DefaultClient}
}

func (d *Driver) Greeting(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.baseURL+"/greeting", nil)
	if err != nil {
		return "", err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var body struct {
		Greeting string `json:"greeting"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}

	return body.Greeting, nil
}
