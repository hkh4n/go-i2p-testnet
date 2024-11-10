package i2pd

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-i2p/go-i2p/lib/common/base64"
	"os"
	"path/filepath"
)

func SyncNetDbToShared(cli *client.Client, ctx context.Context, containerID string, volumeName string) error {
	// Define the source path inside the container
	sourcePath := "/root/.i2pd/netDb"

	// Create a temporary helper container with the shared volume mounted
	helperContainerName := "helper-container"
	helperConfig := &container.Config{
		Image: "alpine",                // Use a lightweight image
		Cmd:   []string{"sleep", "60"}, // Keep the container running for the duration of the operation
	}
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/shared", volumeName),
		},
	}

	resp, err := cli.ContainerCreate(ctx, helperConfig, hostConfig, nil, nil, helperContainerName)
	if err != nil {
		return fmt.Errorf("error creating helper container: %v", err)
	}
	helperContainerID := resp.ID
	defer func() {
		// Clean up the helper container
		removeOptions := container.RemoveOptions{Force: true}
		cli.ContainerRemove(ctx, helperContainerID, removeOptions)
	}()

	// Start the helper container
	startOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, helperContainerID, startOptions); err != nil {
		return fmt.Errorf("error starting helper container: %v", err)
	}

	// Copy the netDb directory from the target container
	reader, _, err := cli.CopyFromContainer(ctx, containerID, sourcePath)
	if err != nil {
		return fmt.Errorf("error copying from container: %v", err)
	}
	defer reader.Close()

	// Copy the netDb directory to the helper container (which has the shared volume mounted)
	err = cli.CopyToContainer(ctx, helperContainerID, "/shared/netDb", reader, types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	if err != nil {
		return fmt.Errorf("error copying to helper container: %v", err)
	}

	fmt.Println("Successfully synchronized netDb to shared volume")
	return nil
}

func SyncSharedToNetDb(cli *client.Client, ctx context.Context, containerID string, volumeName string) error {
	// Define the destination path inside the target container
	destinationPath := "/root/.i2pd/netDb"

	// Create a temporary helper container with the shared volume mounted
	helperContainerName := "helper-container"
	helperConfig := &container.Config{
		Image: "alpine",
		Cmd:   []string{"sleep", "60"},
	}
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/shared", volumeName),
		},
	}

	resp, err := cli.ContainerCreate(ctx, helperConfig, hostConfig, nil, nil, helperContainerName)
	if err != nil {
		return fmt.Errorf("error creating helper container: %v", err)
	}
	helperContainerID := resp.ID
	defer func() {
		removeOptions := container.RemoveOptions{Force: true}
		cli.ContainerRemove(ctx, helperContainerID, removeOptions)
	}()

	// Start the helper container
	startOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, helperContainerID, startOptions); err != nil {
		return fmt.Errorf("error starting helper container: %v", err)
	}

	// Copy the netDb directory from the helper container (shared volume) to the target container
	reader, _, err := cli.CopyFromContainer(ctx, helperContainerID, "/shared/netDb")
	if err != nil {
		return fmt.Errorf("error copying from helper container: %v", err)
	}
	defer reader.Close()

	// Copy the netDb directory to the target container
	copyToContainerOptions := container.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	}
	err = cli.CopyToContainer(ctx, containerID, destinationPath, reader, copyToContainerOptions)
	if err != nil {
		return fmt.Errorf("error copying to target container: %v", err)
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
