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
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

func SyncNetDbToShared(cli *client.Client, ctx context.Context, containerID string, volumeName string) error {
	routerInfoString, filename, directory, err := GetRouterInfoWithFilenameRaw(cli, ctx, containerID)
	if err != nil {
		log.WithError(err).Error("GetRouterInfoWithFilenameRaw failed")
	}
	log.WithFields(logrus.Fields{
		"routerInfoString len": len(routerInfoString),
		"filename":             filename,
		"directory":            directory,
	}).Debug("GetRouterInfoWithFilename Raw")

	// Create the temporary helper container
	helperContainerName := fmt.Sprintf("helper-container-%s", containerID[:12])
	helperConfig := &container.Config{
		Image: "alpine",
		Cmd:   []string{"sleep", "60"},
	}

	hostConfig := container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/shared", volumeName),
		},
	}

	resp, err := cli.ContainerCreate(ctx, helperConfig, &hostConfig, nil, nil, helperContainerName)
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
	execConfig := container.ExecOptions{
		Cmd:          []string{"mkdir", "-p", fmt.Sprintf("/shared/netDb/%s", directory)},
		AttachStdout: true,
		AttachStderr: true,
	}
	execIDResp, err := cli.ContainerExecCreate(ctx, helperContainerID, execConfig)
	if err != nil {
		return fmt.Errorf("error creating mkdir exec: %v", err)
	}

	if err := cli.ContainerExecStart(ctx, execIDResp.ID, types.ExecStartCheck{}); err != nil {
		return fmt.Errorf("error executing mkdir: %v", err)
	}
	// Create a tar archive containing the router info file
	tarBuffer := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuffer)

	header := &tar.Header{
		Name: filename,
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

	// Copy the tar archive to the correct directory in the helper container
	targetPath := fmt.Sprintf("/shared/netDb/%s", directory)
	err = cli.CopyToContainer(ctx, helperContainerID, targetPath, bytes.NewReader(tarBuffer.Bytes()), types.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("error copying router info to helper container: %v", err)
	}

	log.WithFields(logrus.Fields{
		"directory": directory,
		"filename":  filename,
		"path":      targetPath,
	}).Debug("Successfully synced router info to shared volume")
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
