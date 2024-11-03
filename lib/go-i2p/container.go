package go_i2p

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"go-i2p-testnet/lib/docker_control"
)

// createRouterContainer sets up a router container with its configuration.
func CreateRouterContainer(cli *client.Client, ctx context.Context, routerID int, ip string, networkName string, configData string) (string, string, error) {
	containerName := fmt.Sprintf("router-goi2p-%d", routerID)

	log.WithFields(map[string]interface{}{
		"routerID":      routerID,
		"containerName": containerName,
		"ip":            ip,
		"networkName":   networkName,
	}).Debug("Starting router container creation")

	// Create a temporary volume for the configuration
	volumeName := fmt.Sprintf("router%d_config", routerID)
	createOptions := volume.CreateOptions{
		Name: volumeName,
	}

	log.WithFields(map[string]interface{}{
		"volumeName": volumeName,
		"routerID":   routerID,
	}).Debug("Creating configuration volume")

	_, err := cli.VolumeCreate(ctx, createOptions)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"volumeName": volumeName,
			"error":      err,
		}).Error("Failed to create volume")
		return "", "", fmt.Errorf("error creating volume: %v", err)
	}

	// Copy the configuration data into the volume
	log.WithField("volumeName", volumeName).Debug("Copying configuration to volume")
	err = CopyConfigToVolume(cli, ctx, volumeName, configData)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"volumeName": volumeName,
			"error":      err,
		}).Error("Failed to copy config to volume")
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
			fmt.Sprintf("%s:/root", volumeName), // Mount at /.go-i2p
			fmt.Sprintf("%s:/shared", docker_control.SHARED_VOLUME),
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

	log.WithFields(map[string]interface{}{
		"containerName": containerName,
		"image":         containerConfig.Image,
		"volumes":       hostConfig.Binds,
		"ip":            ip,
		"networkName":   networkName,
	}).Debug("Creating container with config")

	// Create the container
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"containerName": containerName,
			"error":         err,
		}).Error("Failed to create container")
		return volumeName, "", fmt.Errorf("error creating container: %v", err)
	}

	// Start the container
	log.WithFields(map[string]interface{}{
		"containerID":   resp.ID,
		"containerName": containerName,
	}).Debug("Starting container")
	startOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, resp.ID, startOptions); err != nil {
		log.WithFields(map[string]interface{}{
			"containerID":   resp.ID,
			"containerName": containerName,
			"error":         err,
		}).Error("Failed to start container")
		return volumeName, "", fmt.Errorf("error starting container: %v", err)
	}

	log.WithFields(map[string]interface{}{
		"containerID":   resp.ID,
		"volumeName":    volumeName,
		"containerName": containerName,
	}).Debug("Successfully created and started router container")

	return resp.ID, volumeName, nil
}
