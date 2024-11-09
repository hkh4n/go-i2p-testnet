package docker_control

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"io"
)

/// /root/.i2pd/router.info

// ReadFileFromContainer reads a file from inside a container and returns its contents
func ReadFileFromContainer(cli *client.Client, ctx context.Context, containerID string, filePath string) (string, error) {
	// Create a reader for the file content
	reader, stat, err := cli.CopyFromContainer(ctx, containerID, filePath)
	if err != nil {
		return "", fmt.Errorf("error copying from container: %v", err)
	}
	defer reader.Close()

	// If it's not a regular file, return error
	if !stat.Mode.IsRegular() {
		return "", fmt.Errorf("requested path is not a regular file")
	}

	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("error reading file content: %v", err)
	}

	return string(content), nil
}
