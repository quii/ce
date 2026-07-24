//go:build integration

package container_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

// apiBaseURL is set once by TestMain and read by every test in this
// package - see TestMain's comment for why this is shared rather than
// per-test.
var apiBaseURL string

func TestGreeting(t *testing.T) {
	driver := container.New(apiBaseURL)

	specifications.GreetingSpecification(t, driver)
}

func TestStartConversation(t *testing.T) {
	driver := container.New(apiBaseURL)

	specifications.ConversationSpecification(t, driver)
}

// TestStartConversation_Projection proves rules 5-7/10 of the "start a
// conversation" story - the pending/resolved 202/after= contract - against
// the real deployed shape: a real relay container polling a real Postgres
// outbox, not just the in-process driver's synchronous Drain.
func TestStartConversation_Projection(t *testing.T) {
	driver := container.New(apiBaseURL)

	specifications.ConversationProjectionSpecification(t, driver)
}

func TestReplyToThread(t *testing.T) {
	driver := container.New(apiBaseURL)

	specifications.ReplyToThreadSpecification(t, driver)
}

func TestAddThread(t *testing.T) {
	driver := container.New(apiBaseURL)

	specifications.AddThreadSpecification(t, driver)
}

func TestAPIDocs(t *testing.T) {
	t.Run("serves the OpenAPI spec", func(t *testing.T) {
		body := get(t, apiBaseURL+"/openapi.yaml")
		assert.True(t, strings.Contains(body, "openapi:"), "response from /openapi.yaml does not contain %q:\n%s", "openapi:", body)
	})

	t.Run("serves the docs UI", func(t *testing.T) {
		body := get(t, apiBaseURL+"/docs")
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

const databaseURL = "postgres://ce:ce@postgres:5432/ce?sslmode=disable"

// TestMain starts one topology - Postgres, the api role and the relay
// role, on a shared Docker network - for every test in this package to
// share, instead of each test bootstrapping (and immediately tearing
// down) its own from scratch. That per-test bootstrap cost ~10-15s
// (sequential container starts/builds plus readiness waits) and dominated
// both `go tool mage test`'s wall time and, far worse, every mutation-
// testing run that needed to re-execute one of these tests per mutant
// (docs/adr/0020-mutation-testing.md) - 6 tests x ~10-15s each, repeated
// per mutant, added up to the majority of a ~40 minute mutate run.
//
// Sharing one Postgres across every test in this file is safe: every
// specification here targets conversations/threads by their own freshly
// server-generated IDs, none of them run in parallel (no t.Parallel in
// this package), and none assert against "all conversations" or an
// absolute sequence number - only against the specific IDs/sequences
// their own writes returned. This is the same reasoning
// internal/adapters/postgres/store_test.go already relies on for sharing
// one container across its own multiple contract-test suites.
func TestMain(m *testing.M) {
	os.Exit(runWithTopology(m))
}

// runWithTopology is split out from TestMain so defer can guarantee every
// container/network gets torn down even if an earlier step fails partway
// through startup - os.Exit (which TestMain must eventually call) does
// not run deferred functions, so nothing in this function's own body ever
// calls it directly.
func runWithTopology(m *testing.M) int {
	ctx := context.Background()

	net, err := network.New(ctx)
	if err != nil {
		log.Printf("create docker network: %v", err)
		return 1
	}
	defer func() { _ = net.Remove(ctx) }()

	stopPostgres, err := startPostgres(ctx, net.Name)
	if err != nil {
		log.Printf("start postgres: %v", err)
		return 1
	}
	defer stopPostgres()

	baseURL, stopAPI, err := startService(ctx, net.Name, "api", true,
		wait.ForHTTP("/greeting").WithPort("8080/tcp").WithStartupTimeout(2*time.Minute))
	if err != nil {
		log.Printf("start api: %v", err)
		return 1
	}
	defer stopAPI()
	apiBaseURL = baseURL

	_, stopRelay, err := startService(ctx, net.Name, "relay", false,
		wait.ForLog("starting relay").WithStartupTimeout(2*time.Minute))
	if err != nil {
		log.Printf("start relay: %v", err)
		return 1
	}
	defer stopRelay()

	return m.Run()
}

func startPostgres(ctx context.Context, networkName string) (stop func(), err error) {
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

	_, stop, err = startContainer(ctx, req)
	return stop, err
}

// startService starts a ce role (api, relay, ...) built from the
// repository's own Dockerfile via its SERVICE build arg, on networkName,
// wired to the shared Postgres startPostgres started. Returns the
// service's externally-reachable base URL when exposeHTTP is true, empty
// otherwise (the relay role serves no HTTP at all).
func startService(ctx context.Context, networkName, service string, exposeHTTP bool, waitingFor wait.Strategy) (baseURL string, stop func(), err error) {
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

	c, stop, err := startContainer(ctx, req)
	if err != nil {
		return "", nil, fmt.Errorf("start %s container: %w", service, err)
	}

	if !exposeHTTP {
		return "", stop, nil
	}

	host, err := c.Host(ctx)
	if err != nil {
		return "", stop, fmt.Errorf("get %s container host: %w", service, err)
	}
	port, err := c.MappedPort(ctx, "8080")
	if err != nil {
		return "", stop, fmt.Errorf("get %s mapped port: %w", service, err)
	}

	return fmt.Sprintf("http://%s:%s", host, port.Port()), stop, nil
}

func startContainer(ctx context.Context, req testcontainers.ContainerRequest) (testcontainers.Container, func(), error) {
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("start container (image %q, dockerfile %q): %w", req.Image, req.Dockerfile, err)
	}

	stop := func() {
		if err := c.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %v", err)
		}
	}

	return c, stop, nil
}
