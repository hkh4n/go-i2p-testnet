package docker_control

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func CreateDockerNetwork(cli *client.Client, ctx context.Context, networkName string) (string, error) {
	// Check if the network already exists
	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, net := range networks {
		if net.Name == networkName {
			fmt.Printf("Network %s already exists. Using existing network.\n", networkName)
			return net.ID, nil
		}
	}

	// Create the network
	createOptions := network.CreateOptions{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet: "172.28.0.0/16",
				},
			},
		},
	}
	resp, err := cli.NetworkCreate(ctx, networkName, createOptions)
	if err != nil {
		return "", err
	}
	fmt.Printf("Created network %s with ID %s\n", networkName, resp.ID)
	return resp.ID, nil
}
