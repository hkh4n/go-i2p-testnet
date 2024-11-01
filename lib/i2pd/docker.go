package i2pd

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"go-i2p-testnet/lib/docker_control"
)

func BuildImage(cli *client.Client, ctx context.Context) error {
	dockerfilePath := "../docker/i2pd-node.dockerfile"
	log.WithFields(map[string]interface{}{
		"imageName":  I2PD_IMAGE,
		"dockerfile": dockerfilePath,
	}).Debug("Starting i2pd Docker image build")
	err := docker_control.BuildDockerImage(cli, ctx, I2PD_IMAGE, dockerfilePath)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"imageName":  I2PD_IMAGE,
			"dockerfile": dockerfilePath,
			"error":      err,
		}).Error("Failed to build i2pd Docker image")
		return fmt.Errorf("error building i2pd Docker image: %v", err)
	}

	log.WithField("imageName", I2PD_IMAGE).Debug("Successfully built i2pd Docker image")
	return nil
}

func RemoveImage(cli *client.Client, ctx context.Context) error {
	log.WithField("imageName", I2PD_IMAGE).Debug("Starting i2pd Docker image removal")
	err := docker_control.RemoveDockerImage(cli, ctx, I2PD_IMAGE)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"imageName": I2PD_IMAGE,
			"error":     err,
		}).Error("Failed to remove i2pd Docker image")
		return fmt.Errorf("error removing i2pd Docker image: %v", err)
	}

	log.WithField("imageName", I2PD_IMAGE).Debug("Successfully removed i2pd Docker image")
	return nil
}
