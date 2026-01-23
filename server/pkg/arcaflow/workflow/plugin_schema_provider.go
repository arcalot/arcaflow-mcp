package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultSchemaTimeout = 30 * time.Second
)

// ContainerPluginSchemaProvider executes plugin containers to get JSON schema.
type ContainerPluginSchemaProvider struct {
	runtime string
	timeout time.Duration
}

// NewContainerPluginSchemaProvider constructs a container-based provider.
func NewContainerPluginSchemaProvider() *ContainerPluginSchemaProvider {
	return &ContainerPluginSchemaProvider{
		runtime: detectContainerRuntime(),
		timeout: defaultSchemaTimeout,
	}
}

// WithRuntime overrides the container runtime binary.
func (p *ContainerPluginSchemaProvider) WithRuntime(runtime string) *ContainerPluginSchemaProvider {
	if strings.TrimSpace(runtime) != "" {
		p.runtime = runtime
	}
	return p
}

// WithTimeout overrides the schema retrieval timeout.
func (p *ContainerPluginSchemaProvider) WithTimeout(timeout time.Duration) *ContainerPluginSchemaProvider {
	if timeout > 0 {
		p.timeout = timeout
	}
	return p
}

// InputJSONSchema returns the plugin input JSON schema for a step.
func (p *ContainerPluginSchemaProvider) InputJSONSchema(
	ctx context.Context,
	image string,
	stepID string,
) (json.RawMessage, error) {
	if p.runtime == "" {
		return nil, fmt.Errorf("no container runtime available")
	}
	args := []string{"run", "--rm", image, "--json-schema", "input"}
	if strings.TrimSpace(stepID) != "" {
		args = append(args, "--step", stepID)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, p.runtime, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("schema command failed: %w: %s", err, stderr.String())
	}
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return nil, fmt.Errorf("empty schema output")
	}
	if !json.Valid([]byte(output)) {
		return nil, fmt.Errorf("schema output is not JSON")
	}
	return json.RawMessage(output), nil
}

func detectContainerRuntime() string {
	candidates := []string{"podman", "docker"}
	for _, name := range candidates {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}
