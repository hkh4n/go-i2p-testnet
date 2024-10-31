package docker_control

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

func CreateSharedVolume(cli *client.Client, ctx context.Context) (string, error) {
	volumeName := "go-i2p-shared"
	createOptions := volume.CreateOptions{
		Name: volumeName,
	}
	_, err := cli.VolumeCreate(ctx, createOptions)
	if err != nil {
		return "", fmt.Errorf("error creating shared volume: %v", err)
	}
	return volumeName, nil
}
