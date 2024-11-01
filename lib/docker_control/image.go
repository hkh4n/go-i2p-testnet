package docker_control

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"go-i2p-testnet/lib/utils/logger"
	"io"
	"path/filepath"
)

var log = logger.GetTestnetLogger()

func BuildDockerImage(cli *client.Client, ctx context.Context, imageName string, dockerfilePath string) error {
	log.WithFields(map[string]interface{}{
		"imageName":  imageName,
		"dockerfile": dockerfilePath,
	}).Debug("Starting Docker image build")
	// Check if the image already exists
	exists, err := imageExists(cli, ctx, imageName)
	if err != nil {
		log.WithError(err).Error("Failed to check if Docker image exists")
		return fmt.Errorf("error checking for Docker image: %v", err)
	}

	// If the image exists, skip building it
	if exists {
		log.WithField("imageName", imageName).Debug("Docker image already exists, skipping build")
		return nil
	}

	// Use the directory of the Dockerfile for creating the tar archive
	dockerfileDir := filepath.Dir(dockerfilePath)
	dockerfileName := filepath.Base(dockerfilePath)

	log.WithFields(map[string]interface{}{
		"dockerfileDir":  dockerfileDir,
		"dockerfileName": dockerfileName,
	}).Debug("Creating tar archive of Dockerfile")

	dockerfileTar, err := archive.TarWithOptions(dockerfileDir, &archive.TarOptions{})
	if err != nil {
		log.WithError(err).Error("Failed to create tar archive of Dockerfile")
		return fmt.Errorf("error creating tar archive of Dockerfile: %v", err)
	}

	log.Debug("Initiating Docker image build")
	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: dockerfileName, // Set custom Dockerfile name here
		Remove:     true,
	}

	resp, err := cli.ImageBuild(ctx, dockerfileTar, buildOptions)
	if err != nil {
		log.WithError(err).Error("Docker image build failed")
		return fmt.Errorf("error building Docker image: %v", err)
	}
	defer resp.Body.Close()

	// Read and print the build output
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		log.WithError(err).Error("Failed to read build output")
		return fmt.Errorf("error reading build output: %v", err)
	}

	log.WithField("output", buf.String()).Debug("Docker build completed")
	return nil
}

// removeDockerImage removes a Docker image by name or ID
func RemoveDockerImage(cli *client.Client, ctx context.Context, imageName string) error {
	log.WithField("imageName", imageName).Debug("Attempting to remove Docker image")

	exists, err := imageExists(cli, ctx, imageName)
	if err != nil {
		log.WithError(err).Error("Failed to check if Docker image exists")
		return err
	}
	if !exists {
		log.WithField("imageName", imageName).Error("Cannot remove non-existent image")
		return fmt.Errorf("error: cant remove image '%s' that doesn't exist", imageName)
	}
	removeOptions := image.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	}

	log.Debug("Removing Docker image")
	removedImages, err := cli.ImageRemove(ctx, imageName, removeOptions)
	if err != nil {
		log.WithError(err).Error("Failed to remove Docker image")
		return fmt.Errorf("error removing image %s: %v", imageName, err)
	}

	// Display removed images
	for _, image := range removedImages {
		log.WithField("imageID", image.Deleted).Debug("Successfully removed image")
	}
	return nil
}
func imageExists(cli *client.Client, ctx context.Context, imageName string) (bool, error) {
	log.WithField("imageName", imageName).Debug("Checking if Docker image exists")

	ListOptions := image.ListOptions{}
	images, err := cli.ImageList(ctx, ListOptions)
	if err != nil {
		log.WithError(err).Error("Failed to list Docker images")
		return false, err
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == imageName || tag == imageName+":latest" {
				log.WithField("imageName", imageName).Debug("Docker image found")
				return true, nil
			}
		}
	}

	log.WithField("imageName", imageName).Debug("Docker image not found")
	return false, nil
}
