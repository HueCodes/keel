package transforms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// RegistryClient is an interface for fetching image digests from registries
type RegistryClient interface {
	// GetDigest fetches the digest for an image:tag
	GetDigest(ctx context.Context, image, tag string) (string, error)
}

// PinImageTagTransform pins unpinned base image tags with sha256 digests
type PinImageTagTransform struct {
	// Client is an optional registry client for fetching digests
	// If nil, no network calls are made and unpinned images are skipped
	Client RegistryClient

	// Timeout for registry requests
	Timeout time.Duration
}

func (t *PinImageTagTransform) Name() string {
	return "pin-image-tag"
}

func (t *PinImageTagTransform) Description() string {
	return "Pin base image tags with sha256 digests for reproducible builds"
}

func (t *PinImageTagTransform) Rules() []string {
	return []string{"SEC003"}
}

func (t *PinImageTagTransform) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	// If no client configured, we can't fetch digests
	if t.Client == nil {
		return false
	}

	changed := false
	timeout := t.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for _, stage := range df.Stages {
		from := stage.From
		if from == nil {
			continue
		}

		// Skip if already pinned with digest
		if from.Digest != "" {
			continue
		}

		// Skip scratch and variable-based images
		if from.Image == "scratch" || strings.HasPrefix(from.Image, "$") {
			continue
		}

		// Skip stage references (FROM builder AS ...)
		if isStageReference(df, from.Image) {
			continue
		}

		// Get the tag to pin (default to "latest")
		tag := from.Tag
		if tag == "" {
			tag = "latest"
		}

		// Fetch the digest from the registry
		digest, err := t.Client.GetDigest(ctx, from.Image, tag)
		if err != nil {
			// Failed to fetch digest, skip this image
			continue
		}

		// Pin the image with the digest
		from.Digest = digest
		changed = true
	}

	return changed
}

// isStageReference checks if an image name refers to a build stage
func isStageReference(df *parser.Dockerfile, image string) bool {
	for _, stage := range df.Stages {
		if stage.Name != "" && strings.EqualFold(stage.Name, image) {
			return true
		}
	}
	return false
}

// DockerHubClient is a RegistryClient implementation for Docker Hub
type DockerHubClient struct {
	HTTPClient *http.Client
}

// NewDockerHubClient creates a new Docker Hub registry client
func NewDockerHubClient() *DockerHubClient {
	return &DockerHubClient{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetDigest fetches the digest for an image from Docker Hub
func (c *DockerHubClient) GetDigest(ctx context.Context, image, tag string) (string, error) {
	// Normalize image name (add library/ prefix for official images)
	if !strings.Contains(image, "/") {
		image = "library/" + image
	}

	// Get authentication token
	tokenURL := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", image)
	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token: %s", resp.Status)
	}

	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	// Fetch manifest to get digest
	manifestURL := fmt.Sprintf("https://registry-1.docker.io/v2/%s/manifests/%s", image, tag)
	req, err = http.NewRequestWithContext(ctx, "HEAD", manifestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+tokenResp.Token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")

	resp, err = c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get manifest: %s", resp.Status)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", fmt.Errorf("no digest in response")
	}

	return digest, nil
}
