package container_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/quii/ce/specifications"
	"github.com/quii/ce/specifications/container"
)

func TestGreeting(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "../../",
			Dockerfile: "Dockerfile",
		},
		ExposedPorts: []string{"8080/tcp"},
		Env:          map[string]string{"CE_ROLE": "api"},
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

	driver := container.New(fmt.Sprintf("http://%s:%s", host, port.Port()))

	specifications.GreetingSpecification(t, driver)
}
