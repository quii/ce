package container_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/quii/ce/specifications"
	"github.com/quii/ce/specifications/container"
)

func TestGreeting(t *testing.T) {
	driver := container.New(startCEContainer(t))

	specifications.GreetingSpecification(t, driver)
}

func TestAPIDocs(t *testing.T) {
	baseURL := startCEContainer(t)

	t.Run("serves the OpenAPI spec", func(t *testing.T) {
		body := get(t, baseURL+"/openapi.yaml")
		if want := "openapi:"; !strings.Contains(body, want) {
			t.Errorf("response from /openapi.yaml does not contain %q:\n%s", want, body)
		}
	})

	t.Run("serves the docs UI", func(t *testing.T) {
		body := get(t, baseURL+"/docs")
		if want := "/openapi.yaml"; !strings.Contains(body, want) {
			t.Errorf("response from /docs does not reference %q:\n%s", want, body)
		}
	})
}

func get(t *testing.T, url string) string {
	t.Helper()

	resp, err := http.Get(url) //nolint:gosec,noctx // test-only request to a URL we just built from a locally started container
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d", url, resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body from %s: %v", url, err)
	}

	return string(body)
}

func startCEContainer(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "../../",
			Dockerfile: "Dockerfile",
		},
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForHTTP("/greeting").WithPort("8080/tcp").WithStartupTimeout(2 * time.Minute),
	}

	ceContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start ce container: %v", err)
	}
	t.Cleanup(func() {
		if err := ceContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate ce container: %v", err)
		}
	})

	host, err := ceContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}
	port, err := ceContainer.MappedPort(ctx, "8080")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	return fmt.Sprintf("http://%s:%s", host, port.Port())
}
