package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/go-i2p/go-i2p/lib/config"
	"go-i2p-testnet/lib/docker_control"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
)

// "go-i2p-testnet/lib/docker_control"
// initializeRouterConfig sets up a router-specific configuration for each instance
func initializeRouterConfig(routerID int) *config.RouterConfig {
	// Define base directory for this router's configuration
	baseDir := filepath.Join("testnet", fmt.Sprintf("router%d", routerID))
	err := os.MkdirAll(baseDir, os.ModePerm)
	if err != nil {
		log.Printf("failed to create baseDir: %v", err)
		return nil
	}

	// Assign each router its own netDb and working directory
	netDbPath := filepath.Join(baseDir, "netDb")
	workingDir := filepath.Join(baseDir, "config")
	err = os.MkdirAll(netDbPath, os.ModePerm)
	if err != nil {
		log.Printf("failed to create netDbPath: %v", err)
		return nil
	}
	err = os.MkdirAll(workingDir, os.ModePerm)
	if err != nil {
		log.Printf("failed to create workingDir: %v", err)
		return nil
	}

	// Create and return a RouterConfig instance
	return &config.RouterConfig{
		BaseDir:    baseDir,
		WorkingDir: workingDir,
		NetDb:      &config.NetDbConfig{Path: netDbPath},
		Bootstrap:  &config.DefaultBootstrapConfig, // Modify as needed for custom bootstrap setup
	}
}
func generateRouterConfig(routerID int, ip string, peers []string) string {
	// Initialize router-specific configuration
	routerConfig := initializeRouterConfig(routerID)

	// Define common settings for each router instance
	configData := fmt.Sprintf(`
		netID=12345
		reseed.disable=true
		router.transport.udp.host=%s
		router.transport.udp.port=7654
		netDb.path=%s
	`, ip, routerConfig.NetDb.Path)

	// Add peers to the configuration
	for i, peer := range peers {
		configData += fmt.Sprintf("peer.%d=%s\n", i+1, peer)
	}

	return configData
}
func createDockerNetwork(cli *client.Client, ctx context.Context, networkName string) (string, error) {
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

// createRouterContainer sets up a router container with its configuration.
func createRouterContainer(cli *client.Client, ctx context.Context, routerID int, ip string, networkName string, configData string) (string, string, error) {
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
	err = docker_control.CopyConfigToVolume(cli, ctx, volumeName, configData)
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
	networkID, err := createDockerNetwork(cli, ctx, networkName)
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
		configData := generateRouterConfig(routerID, ip, peers)

		// Create the container
		containerID, volumeName, err := createRouterContainer(cli, ctx, routerID, ip, networkName, configData)
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
