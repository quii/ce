//go:build integration

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

	"github.com/quii/ce/internal/assert"
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

func TestReplyToThread(t *testing.T) {
	driver := container.New(startTopology(t).apiBaseURL)

	specifications.ReplyToThreadSpecification(t, driver)
}

func TestAddThread(t *testing.T) {
	driver := container.New(startTopology(t).apiBaseURL)

	specifications.AddThreadSpecification(t, driver)
}

func TestAPIDocs(t *testing.T) {
	baseURL := startTopology(t).apiBaseURL

	t.Run("serves the OpenAPI spec", func(t *testing.T) {
		body := get(t, baseURL+"/openapi.yaml")
		assert.True(t, strings.Contains(body, "openapi:"), "response from /openapi.yaml does not contain %q:\n%s", "openapi:", body)
	})

	t.Run("serves the docs UI", func(t *testing.T) {
		body := get(t, baseURL+"/docs")
		assert.True(t, strings.Contains(body, "/openapi.yaml"), "response from /docs does not reference %q:\n%s", "/openapi.yaml", body)
	})
}

func get(t *testing.T, url string) string {
	t.Helper()

	resp, err := http.Get(url) //nolint:gosec,noctx // test-only request to a URL we just built from a locally started container
	assert.NoErr(t, err, "GET %s", url)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, resp.StatusCode, http.StatusOK, "GET %s status", url)

	body, err := io.ReadAll(resp.Body)
	assert.NoErr(t, err, "read response body from %s", url)

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
	assert.NoErr(t, err, "create docker network")
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
	assert.NoErr(t, err, "get %s container host", service)
	port, err := c.MappedPort(ctx, "8080")
	assert.NoErr(t, err, "get %s mapped port", service)

	return fmt.Sprintf("http://%s:%s", host, port.Port())
}

func startContainer(t *testing.T, req testcontainers.ContainerRequest) testcontainers.Container {
	t.Helper()
	ctx := context.Background()

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoErr(t, err, "start container (image %q, dockerfile %q)", req.Image, req.Dockerfile)
	t.Cleanup(func() {
		if err := c.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	return c
}
