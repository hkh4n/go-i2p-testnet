package docker_control

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

func CreateSharedVolume(cli *client.Client, ctx context.Context) (string, error) {
	volumeName := "go-i2p-shared"

	log.WithField("volumeName", volumeName).Debug("Starting Docker volume creation")

	createOptions := volume.CreateOptions{
		Name: volumeName,
	}

	log.WithFields(map[string]interface{}{
		"volumeName": volumeName,
		"options":    createOptions,
	}).Debug("Creating Docker volume")

	_, err := cli.VolumeCreate(ctx, createOptions)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"volumeName": volumeName,
			"error":      err,
		}).Error("Failed to create Docker volume")
		return "", fmt.Errorf("error creating shared volume: %v", err)
	}
	log.WithField("volumeName", volumeName).Debug("Successfully created Docker volume")
	return volumeName, nil
}
