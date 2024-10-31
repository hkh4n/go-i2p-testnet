package docker_control

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"io"
	"path/filepath"
)

func BuildDockerImage(cli *client.Client, ctx context.Context, imageName string, dockerfilePath string) error {
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

	// Use the directory of the Dockerfile for creating the tar archive
	dockerfileDir := filepath.Dir(dockerfilePath)
	dockerfileName := filepath.Base(dockerfilePath)

	dockerfileTar, err := archive.TarWithOptions(dockerfileDir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("error creating tar archive of Dockerfile: %v", err)
	}

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: dockerfileName, // Set custom Dockerfile name here
		Remove:     true,
	}

	resp, err := cli.ImageBuild(ctx, dockerfileTar, buildOptions)
	if err != nil {
		return fmt.Errorf("error building Docker image: %v", err)
	}
	defer resp.Body.Close()

	// Read and print the build output
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return fmt.Errorf("error reading build output: %v", err)
	}
	fmt.Println("Docker build output:", buf.String())

	return nil
}

// removeDockerImage removes a Docker image by name or ID
func removeDockerImage(cli *client.Client, ctx context.Context, imageName string) error {
	exists, err := imageExists(cli, ctx, imageName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("error: cant remove image '%s' that doesn't exist", imageName)
	}
	removeOptions := image.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	}

	removedImages, err := cli.ImageRemove(ctx, imageName, removeOptions)
	if err != nil {
		return fmt.Errorf("error removing image %s: %v", imageName, err)
	}

	// Display removed images
	for _, image := range removedImages {
		fmt.Printf("Removed image: %s\n", image.Deleted)
	}
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