package go_i2p

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"go-i2p-testnet/lib/docker_control"
)

const GOI2P_IMAGE = "go-i2p-node"

func BuildImage(cli *client.Client, ctx context.Context) error {
	dockerfilePath := "../docker/go-i2p-node.dockerfile"
	log.WithFields(map[string]interface{}{
		"imageName":  GOI2P_IMAGE,
		"dockerfile": dockerfilePath,
	}).Debug("Starting Docker image build")
	err := docker_control.BuildDockerImage(cli, ctx, GOI2P_IMAGE, dockerfilePath)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"imageName":  GOI2P_IMAGE,
			"dockerfile": dockerfilePath,
			"error":      err,
		}).Error("Failed to build Docker image")
		return fmt.Errorf("error building Docker image: %v", err)
	}

	log.WithField("imageName", GOI2P_IMAGE).Debug("Successfully built Docker image")
	return nil
}

func RemoveImage(cli *client.Client, ctx context.Context) error {
	log.WithField("imageName", GOI2P_IMAGE).Debug("Starting Docker image removal")
	err := docker_control.RemoveDockerImage(cli, ctx, "go-i2p-node")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"imageName": GOI2P_IMAGE,
			"error":     err,
		}).Error("Failed to remove Docker image")
		return fmt.Errorf("error removing Docker image: %v", err)
	}

	log.WithField("imageName", GOI2P_IMAGE).Debug("Successfully removed Docker image")
	return nil
}
