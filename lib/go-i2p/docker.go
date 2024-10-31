package go_i2p

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"go-i2p-testnet/lib/docker_control"
)

func BuildImage(cli *client.Client, ctx context.Context) error {
	err := docker_control.BuildDockerImage(cli, ctx, "go-i2p-node", "../docker/go-i2p-node.dockerfile")
	if err != nil {
		return fmt.Errorf("error building Docker image: %v", err)
	}
	return nil
}

func RemoveImage(cli *client.Client, ctx context.Context) error {
	err := docker_control.RemoveDockerImage(cli, ctx, "go-i2p-node")
	if err != nil {
		return fmt.Errorf("error removing Docker image: %v", err)
	}
	return nil
}
