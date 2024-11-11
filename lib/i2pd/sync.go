package i2pd

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-i2p/go-i2p/lib/common/base64"
	"io"
	"os"
)

func SyncNetDbToShared(cli *client.Client, ctx context.Context, containerID string, volumeName string) error {
	sourcePath := "/root/.i2pd/netDb/"

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

	startOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, helperContainerID, startOptions); err != nil {
		return fmt.Errorf("error starting helper container: %v", err)
	}

	// First ensure temp and target directories exist and are clean
	execConfig := types.ExecConfig{
		Cmd:          []string{"sh", "-c", "rm -rf /shared/netDb /tmp/netdb_extract && mkdir -p /shared/netDb /tmp/netdb_extract"},
		AttachStdout: true,
		AttachStderr: true,
	}
	execIDResp, err := cli.ContainerExecCreate(ctx, helperContainerID, execConfig)
	if err != nil {
		return fmt.Errorf("error creating exec config: %v", err)
	}
	if err := cli.ContainerExecStart(ctx, execIDResp.ID, types.ExecStartCheck{}); err != nil {
		return fmt.Errorf("error starting exec: %v", err)
	}

	// Copy from source container to temp location
	reader, _, err := cli.CopyFromContainer(ctx, containerID, sourcePath)
	if err != nil {
		return fmt.Errorf("error copying from container: %v", err)
	}
	defer reader.Close()

	err = cli.CopyToContainer(ctx, helperContainerID, "/tmp/netdb_extract", reader, types.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("error copying to temp: %v", err)
	}

	// Move contents to final location
	execConfig = types.ExecConfig{
		Cmd: []string{"sh", "-c", "cp -r /tmp/netdb_extract/root/.i2pd/netDb/* /shared/netDb/ 2>/dev/null || true"},
	}
	execIDResp, err = cli.ContainerExecCreate(ctx, helperContainerID, execConfig)
	if err != nil {
		return fmt.Errorf("error moving files: %v", err)
	}
	if err := cli.ContainerExecStart(ctx, execIDResp.ID, types.ExecStartCheck{}); err != nil {
		return fmt.Errorf("error moving files: %v", err)
	}

	// Clean up temp directory
	execConfig = types.ExecConfig{
		Cmd: []string{"rm", "-rf", "/tmp/netdb_extract"},
	}
	execIDResp, err = cli.ContainerExecCreate(ctx, helperContainerID, execConfig)
	if err != nil {
		return fmt.Errorf("error cleaning up: %v", err)
	}
	if err := cli.ContainerExecStart(ctx, execIDResp.ID, types.ExecStartCheck{}); err != nil {
		return fmt.Errorf("error cleaning up: %v", err)
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

	// **Check if /shared/netDb exists in the helper container**
	execConfig := types.ExecConfig{
		Cmd:          []string{"ls", "/shared/netDb"},
		AttachStdout: true,
		AttachStderr: true,
	}
	execIDResp, err := cli.ContainerExecCreate(ctx, helperContainerID, execConfig)
	if err != nil {
		return fmt.Errorf("error creating exec config: %v", err)
	}
	err = cli.ContainerExecStart(ctx, execIDResp.ID, types.ExecStartCheck{})
	if err != nil {
		return fmt.Errorf("error starting exec: %v", err)
	}

	// Copy the netDb directory from the helper container (shared volume) to the target container
	reader, _, err := cli.CopyFromContainer(ctx, helperContainerID, "/shared/netDb")
	if err != nil {
		return fmt.Errorf("error copying from helper container: %v", err)
	}
	defer reader.Close()

	// Ensure the destination directory exists in the router container
	execConfig = types.ExecConfig{
		Cmd:          []string{"mkdir", "-p", destinationPath},
		AttachStdout: true,
		AttachStderr: true,
	}
	execIDResp, err = cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("error creating exec config: %v", err)
	}
	err = cli.ContainerExecStart(ctx, execIDResp.ID, types.ExecStartCheck{})
	if err != nil {
		return fmt.Errorf("error starting exec: %v", err)
	}

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
func SyncRouterInfoToNetDb(cli *client.Client, ctx context.Context, containerID string, volumeName string) error {
	// Get RouterInfo, routerInfoString, and the generated filename
	ri, routerInfoString, filename, err := GetRouterInfoWithFilename(cli, ctx, containerID)
	if err != nil {
		return fmt.Errorf("error getting router info: %v", err)
	}
	log.Debugf("got filename: %s\n", filename)

	// Extract the first two characters of the encoded hash
	identHash := ri.IdentHash()
	encodedHash := base64.EncodeToString(identHash[:])
	directory := encodedHash[:2] // Get the first two characters

	// Create a temporary helper container with the shared volume mounted
	helperContainerName := fmt.Sprintf("helper-container-%s", containerID[:12])
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
		// Clean up the helper container
		removeOptions := container.RemoveOptions{Force: true}
		cli.ContainerRemove(ctx, helperContainerID, removeOptions)
	}()

	// Start the helper container
	startOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, helperContainerID, startOptions); err != nil {
		return fmt.Errorf("error starting helper container: %v", err)
	}

	// Define the target directory inside the helper container
	targetDir := "/shared/netDb"

	// Create a tar archive with the directory and file
	tarBuffer := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuffer)
	// Include the directory in the tar header
	header := &tar.Header{
		Name: fmt.Sprintf("%s/%s", directory, filename), // Include directory
		Mode: 0600,
		Size: int64(len(routerInfoString)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("error writing tar header: %v", err)
	}
	if _, err := tw.Write([]byte(routerInfoString)); err != nil {
		return fmt.Errorf("error writing router info to tar: %v", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("error closing tar writer: %v", err)
	}

	// Copy the tar archive to the helper container
	err = cli.CopyToContainer(ctx, helperContainerID, targetDir, bytes.NewReader(tarBuffer.Bytes()), types.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("error copying router info to helper container: %v", err)
	}

	// Optionally, list the contents of the target directory to verify
	execConfig := types.ExecConfig{
		Cmd:          []string{"ls", "-lR", targetDir},
		AttachStdout: true,
		AttachStderr: true,
	}
	execIDResp, err := cli.ContainerExecCreate(ctx, helperContainerID, execConfig)
	if err != nil {
		return fmt.Errorf("error creating exec config for listing: %v", err)
	}
	respAttach, err := cli.ContainerExecAttach(ctx, execIDResp.ID, types.ExecStartCheck{})
	if err != nil {
		return fmt.Errorf("error attaching to exec: %v", err)
	}
	defer respAttach.Close()
	io.Copy(os.Stdout, respAttach.Reader)

	fmt.Printf("Successfully synced RouterInfo to %s\n", targetDir)
	return nil
}
