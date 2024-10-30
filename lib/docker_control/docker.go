package docker_control

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"io"
	"log"
)

func CopyConfigToVolume(cli *client.Client, ctx context.Context, volumeName string, configData string) error {
	// Create a temporary container to copy data into the volume
	tempContainerConfig := &container.Config{
		Image:      "alpine",
		Tty:        false,
		WorkingDir: "/config",
		Cmd:        []string{"sh", "-c", "sleep 1d"},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/config", volumeName),
		},
	}

	resp, err := cli.ContainerCreate(ctx, tempContainerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("error creating temporary container: %v", err)
	}
	defer func() {
		RemoveOptions := container.RemoveOptions{Force: true}
		err := cli.ContainerRemove(ctx, resp.ID, RemoveOptions)
		if err != nil {
			log.Printf("failed to remove container: %v", err)
		}
	}()

	// Start the container
	StartOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, resp.ID, StartOptions); err != nil {
		return fmt.Errorf("error starting temporary container: %v", err)
	}

	// Copy the configuration file into the container
	tarReader, err := createTarArchive("router.config", configData)
	if err != nil {
		return fmt.Errorf("error creating tar archive: %v", err)
	}

	// Copy to the container's volume-mounted directory
	err = cli.CopyToContainer(ctx, resp.ID, "/config", tarReader, container.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("error copying to container: %v", err)
	}

	// Stop the container
	StopOptions := container.StopOptions{}
	if err := cli.ContainerStop(ctx, resp.ID, StopOptions); err != nil {
		return fmt.Errorf("error stopping temporary container: %v", err)
	}

	return nil
}
func BuildDockerImage(cli *client.Client, ctx context.Context, imageName string, dockerfileDir string) error {
	// Check if the image already exists
	exists, err := imageExists(cli, ctx, imageName)
	if err != nil {
		return fmt.Errorf("error checking for Docker image: %v", err)
	}

	// If the image exists, skip building it
	if exists {
		fmt.Printf("Docker image %s already exists. Skipping build.\n", imageName)
		return nil
	}

	dockerfileTar, err := archive.TarWithOptions(dockerfileDir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("error creating tar archive of Dockerfile: %v", err)
	}

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
		Remove:     true,
	}

	resp, err := cli.ImageBuild(ctx, dockerfileTar, buildOptions)
	if err != nil {
		return fmt.Errorf("error building Docker image: %v", err)
	}
	defer resp.Body.Close()

	// Read the output
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return fmt.Errorf("error reading build output: %v", err)
	}
	fmt.Println("Docker build output:", buf.String())

	return nil
}
func imageExists(cli *client.Client, ctx context.Context, imageName string) (bool, error) {
	ListOptions := image.ListOptions{}
	images, err := cli.ImageList(ctx, ListOptions)
	if err != nil {
		return false, err
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == imageName || tag == imageName+":latest" {
				return true, nil
			}
		}
	}
	return false, nil
}

func createTarArchive(filename, content string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	hdr := &tar.Header{
		Name: filename,
		Mode: 0600,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf, nil
}
