package container

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

type Driver struct {
	baseURL string
	client  *http.Client
}

func New(baseURL string) *Driver {
	return &Driver{baseURL: baseURL, client: http.DefaultClient}
}

func (d *Driver) Greet(ctx context.Context, cmd in.GetGreetingCommand) (domain.Greeting, error) {
	target := d.baseURL + "/greeting"
	if cmd.Name != "" {
		query := url.Values{}
		query.Add("name", cmd.Name)
		target += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
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

	return domain.Greeting(body.Greeting), nil
}
