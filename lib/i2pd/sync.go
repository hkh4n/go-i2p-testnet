package i2pd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-i2p/go-i2p/lib/common/base64"
	"io"
	"os"
	"path/filepath"
)

func SyncNetDbToShared(cli *client.Client, ctx context.Context, containerID string) error {
	// Define the source and destination paths
	sourcePath := "/root/.i2pd/netDb"
	destinationPath := "/shared/netDb"

	// Use Docker's API to copy files from the container
	reader, _, err := cli.CopyFromContainer(ctx, containerID, sourcePath)
	if err != nil {
		return fmt.Errorf("error copying from container: %v", err)
	}
	defer reader.Close()

	// Read the content from the reader
	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("error reading content: %v", err)
	}

	// Write the content to the shared directory
	sharedDir := filepath.Join(destinationPath)
	err = os.MkdirAll(sharedDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating shared directory: %v", err)
	}

	// Write content to the shared directory
	err = os.WriteFile(filepath.Join(sharedDir, "netDb.tar"), content, 0644)
	if err != nil {
		return fmt.Errorf("error writing to shared directory: %v", err)
	}

	fmt.Println("Successfully synchronized netDb to shared volume")
	return nil
}

func SyncSharedToNetDb(cli *client.Client, ctx context.Context, containerID string) error {
	// Define the source and destination paths
	sourcePath := "/shared/netDb/netDb.tar"
	destinationPath := "/root/.i2pd/netDb"

	// Open the tar file from the shared directory
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("error reading from shared directory: %v", err)
	}

	// Create a tar reader from the content
	tarReader := bytes.NewReader(content)
	tarStream := io.NopCloser(tarReader) // Create a closer for the tar stream

	// Use Docker's API to copy files to the container
	err = cli.CopyToContainer(ctx, containerID, destinationPath, tarStream, container.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("error copying to container: %v", err)
	}

	fmt.Println("Successfully synchronized netDb from shared volume to container")
	return nil
}

// SyncRouterInfoToNetDb sorts the RouterInfo into the proper location in netDb
func SyncRouterInfoToNetDb(cli *client.Client, ctx context.Context, containerID string, netDbPath string) error {
	// Get RouterInfo, routerInfoString, and the generated filename
	ri, routerInfoString, filename, err := GetRouterInfoWithFilename(cli, ctx, containerID)
	if err != nil {
		return fmt.Errorf("error getting router info: %v", err)
	}

	// Extract the first two characters of the encoded hash
	identHash := ri.IdentHash()
	encodedHash := base64.EncodeToString(identHash[:])
	directory := encodedHash[:2] // Get the first two characters

	// Create the directory if it doesn't exist
	targetDir := filepath.Join(netDbPath, directory)
	err = os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating directory %s: %v", targetDir, err)
	}

	// Write the RouterInfo file to the correct location
	targetFilePath := filepath.Join(targetDir, filename)
	err = os.WriteFile(targetFilePath, []byte(routerInfoString), 0644)
	if err != nil {
		return fmt.Errorf("error writing router info file: %v", err)
	}

	fmt.Printf("Successfully synced RouterInfo to %s\n", targetFilePath)
	return nil
}
