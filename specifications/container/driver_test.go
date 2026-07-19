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
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/quii/ce/specifications"
	"github.com/quii/ce/specifications/container"
)

func TestGreeting(t *testing.T) {
	driver := container.New(startTopology(t).apiBaseURL)

	specifications.GreetingSpecification(t, driver)
}

func TestStartConversation(t *testing.T) {
	driver := container.New(startTopology(t).apiBaseURL)

	specifications.ConversationSpecification(t, driver)
}

// TestStartConversation_Projection proves rules 5-7/10 of the "start a
// conversation" story - the pending/resolved 202/after= contract - against
// the real deployed shape: a real relay container polling a real Postgres
// outbox, not just the in-process driver's synchronous Drain.
func TestStartConversation_Projection(t *testing.T) {
	driver := container.New(startTopology(t).apiBaseURL)

	specifications.ConversationProjectionSpecification(t, driver)
}

func TestAPIDocs(t *testing.T) {
	baseURL := startTopology(t).apiBaseURL

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

type topology struct {
	apiBaseURL string
}

const databaseURL = "postgres://ce:ce@postgres:5432/ce?sslmode=disable"

// startTopology brings up the same three roles docker-compose.yml wires
// together in production - Postgres, the api role and the relay role - on
// a shared Docker network, api and relay both pointed at the same
// database. This is what makes the relay's real, asynchronous draining
// (docs/write-path.md) observable through the container driver, not just
// the in-process one.
func startTopology(t *testing.T) topology {
	t.Helper()
	ctx := context.Background()

	net, err := network.New(ctx)
	if err != nil {
		t.Fatalf("failed to create docker network: %v", err)
	}
	t.Cleanup(func() {
		if err := net.Remove(ctx); err != nil {
			t.Logf("failed to remove docker network: %v", err)
		}
	})

	startPostgres(t, net.Name)
	apiBaseURL := startService(t, net.Name, "api", true,
		wait.ForHTTP("/greeting").WithPort("8080/tcp").WithStartupTimeout(2*time.Minute))
	startService(t, net.Name, "relay", false,
		wait.ForLog("starting relay").WithStartupTimeout(2*time.Minute))

	return topology{apiBaseURL: apiBaseURL}
}

func startPostgres(t *testing.T, networkName string) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image: "postgres:17-alpine",
		Env: map[string]string{
			"POSTGRES_USER":     "ce",
			"POSTGRES_PASSWORD": "ce",
			"POSTGRES_DB":       "ce",
		},
		Networks:       []string{networkName},
		NetworkAliases: map[string][]string{networkName: {"postgres"}},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(2 * time.Minute),
	}

	startContainer(t, req)
}

// startService starts a ce role (api, relay, ...) built from the
// repository's own Dockerfile via its SERVICE build arg, on networkName,
// wired to the shared Postgres started by startPostgres. Returns the
// service's externally-reachable base URL when exposeHTTP is true, empty
// otherwise (the relay role serves no HTTP at all).
func startService(t *testing.T, networkName, service string, exposeHTTP bool, waitingFor wait.Strategy) string {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "../../",
			Dockerfile: "Dockerfile",
			BuildArgs:  map[string]*string{"SERVICE": &service},
		},
		Env:        map[string]string{"DATABASE_URL": databaseURL},
		Networks:   []string{networkName},
		WaitingFor: waitingFor,
	}
	if exposeHTTP {
		req.ExposedPorts = []string{"8080/tcp"}
	}

	c := startContainer(t, req)

	if !exposeHTTP {
		return ""
	}

	host, err := c.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get %s container host: %v", service, err)
	}
	port, err := c.MappedPort(ctx, "8080")
	if err != nil {
		t.Fatalf("failed to get %s mapped port: %v", service, err)
	}

	return fmt.Sprintf("http://%s:%s", host, port.Port())
}

func startContainer(t *testing.T, req testcontainers.ContainerRequest) testcontainers.Container {
	t.Helper()
	ctx := context.Background()

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container (image %q, dockerfile %q): %v", req.Image, req.Dockerfile, err)
	}
	t.Cleanup(func() {
		if err := c.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	return c
}
