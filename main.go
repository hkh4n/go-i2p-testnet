package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go-i2p-testnet/lib/docker_control"
	go_i2p "go-i2p-testnet/lib/go-i2p"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// cleanup removes all created Docker resources: containers, volumes, and network.
func cleanup(cli *client.Client, ctx context.Context, createdContainers []string, createdVolumes []string, networkName string) {
	fmt.Println("\nCleaning up Docker resources...")

	// Remove containers
	for _, containerID := range createdContainers {
		// Attempt to stop the container
		timeout := 10 // seconds
		err := cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
		if err != nil {
			log.Printf("Warning: Failed to stop container %s: %v", containerID, err)
		}

		// Attempt to remove the container
		removeOptions := container.RemoveOptions{Force: true}
		err = cli.ContainerRemove(ctx, containerID, removeOptions)
		if err != nil {
			log.Printf("Warning: Failed to remove container %s: %v", containerID, err)
		} else {
			fmt.Printf("Removed container %s\n", containerID)
		}
	}

	// Remove volumes
	for _, volumeName := range createdVolumes {
		err := cli.VolumeRemove(ctx, volumeName, true)
		if err != nil {
			log.Printf("Warning: Failed to remove volume %s: %v", volumeName, err)
		} else {
			fmt.Printf("Removed volume %s\n", volumeName)
		}
	}

	// Remove network
	err := cli.NetworkRemove(ctx, networkName)
	if err != nil {
		log.Printf("Warning: Failed to remove network %s: %v", networkName, err)
	} else {
		fmt.Printf("Removed network %s\n", networkName)
	}
}

func main() {
	ctx := context.Background()

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error creating Docker client: %v", err)
	}

	// Track created containers and volumes for cleanup
	var createdContainers []string
	var createdVolumes []string
	var mu sync.Mutex // To protect access to the slices

	// Function to add container and volume IDs to the tracking slices
	addCreated := func(containerID, volumeID string) {
		mu.Lock()
		defer mu.Unlock()
		createdContainers = append(createdContainers, containerID)
		createdVolumes = append(createdVolumes, volumeID)
	}
	// Ensure cleanup is performed on exit
	defer func() {
		if len(createdContainers) == 0 && len(createdVolumes) == 0 {
			return
		}
		cleanup(cli, ctx, createdContainers, createdVolumes, "go-i2p-testnet")
	}()

	// Set up signal handling to gracefully handle termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	err = docker_control.BuildDockerImage(cli, ctx, "go-i2p-node", "../docker/go-i2p-node.dockerfile")
	if err != nil {
		log.Fatalf("Error building Docker image: %v", err)
	}

	// Create Docker network
	networkName := "go-i2p-testnet"
	networkID, err := docker_control.CreateDockerNetwork(cli, ctx, networkName)
	if err != nil {
		log.Fatalf("Error creating Docker network: %v", err)
	}
	fmt.Printf("Created network %s with ID %s\n", networkName, networkID)

	// Number of routers to create
	numRouters := 3

	// IP addresses for the routers
	baseIP := "172.28.0."
	routerIPs := make([]string, numRouters)
	for i := 0; i < numRouters; i++ {
		routerIPs[i] = fmt.Sprintf("%s%d", baseIP, i+2) // Starting from .2
	}

	// Spin up routers
	for i := 0; i < numRouters; i++ {
		routerID := i + 1
		ip := routerIPs[i]

		// Collect peers (other router IPs)
		peers := make([]string, 0)
		for j, peerIP := range routerIPs {
			if j != i {
				peers = append(peers, peerIP)
			}
		}

		// Generate router configuration
		configData := go_i2p.GenerateRouterConfig(routerID, ip, peers)

		// Create the container
		containerID, volumeName, err := go_i2p.CreateRouterContainer(cli, ctx, routerID, ip, networkName, configData)
		if err != nil {
			log.Fatalf("Error creating router container: %v", err)
		}
		fmt.Printf("Started router%d with IP %s\n", routerID, ip)

		// Track the created container and volume for cleanup
		addCreated(containerID, volumeName)
	}
	// Inform the user that routers are running
	fmt.Println("All routers are up and running. Press Ctrl+C to stop and clean up.")

	// Wait for interrupt signal to gracefully shutdown
	<-sigs
	fmt.Println("\nReceived interrupt signal. Initiating cleanup...")
}
