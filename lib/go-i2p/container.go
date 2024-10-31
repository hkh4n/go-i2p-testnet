package go_i2p

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// createRouterContainer sets up a router container with its configuration.
func CreateRouterContainer(cli *client.Client, ctx context.Context, routerID int, ip string, networkName string, configData string) (string, string, error) {
	containerName := fmt.Sprintf("router%d", routerID)

	// Create a temporary volume for the configuration
	volumeName := fmt.Sprintf("router%d_config", routerID)
	createOptions := volume.CreateOptions{
		Name: volumeName,
	}
	_, err := cli.VolumeCreate(ctx, createOptions)
	if err != nil {
		return "", "", fmt.Errorf("error creating volume: %v", err)
	}

	// Copy the configuration data into the volume
	err = CopyConfigToVolume(cli, ctx, volumeName, configData)
	if err != nil {
		return "", "", fmt.Errorf("error copying config to volume: %v", err)
	}

	// Prepare container configuration
	containerConfig := &container.Config{
		Image: "go-i2p-node",
		Cmd:   []string{"go-i2p"},
	}

	// Host configuration
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/config", volumeName),
		},
	}

	// Network settings
	endpointSettings := &network.EndpointSettings{
		IPAMConfig: &network.EndpointIPAMConfig{
			IPv4Address: ip,
		},
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: endpointSettings,
		},
	}

	// Create the container
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		return volumeName, "", fmt.Errorf("error creating container: %v", err)
	}

	// Start the container
	startOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, resp.ID, startOptions); err != nil {
		return volumeName, "", fmt.Errorf("error starting container: %v", err)
	}

	return resp.ID, volumeName, nil
}
