package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"io"
	"log"
)

func generateRouterConfig(routerID int, ip string, peers []string) string {
	config := fmt.Sprintf(`
	netID=12345
	reseed.disable=true
	router.transport.udp.host=%s
	router.transport.udp.port=7654
	`, ip)

	// Add peers to the configuration
	for i, peer := range peers {
		config += fmt.Sprintf("peer.%d=%s\n", i+1, peer)
	}

	return config
}

func createDockerNetwork(cli *client.Client, ctx context.Context, networkName string) (string, error) {
	// Check if the network already exists
	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
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
	resp, err := cli.NetworkCreate(ctx, networkName, types.NetworkCreate{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet: "172.28.0.0/16",
				},
			},
		},
	})
	if err != nil {
		return "", err
	}
	fmt.Printf("Created network %s with ID %s\n", networkName, resp.ID)
	return resp.ID, nil
}
func createRouterContainer(cli *client.Client, ctx context.Context, routerID int, ip string, networkName string, configData string) error {
	containerName := fmt.Sprintf("router%d", routerID)

	// Create a temporary volume for the configuration
	volumeName := fmt.Sprintf("router%d_config", routerID)
	CreateOptions := volume.CreateOptions{Name: volumeName}
	_, err := cli.VolumeCreate(ctx, CreateOptions)
	if err != nil {
		return fmt.Errorf("Error creating volume: %v", err)
	}

	// Copy the configuration data into the volume
	err = copyConfigToVolume(cli, ctx, volumeName, configData)
	if err != nil {
		return fmt.Errorf("Error copying config to volume: %v", err)
	}

	// Prepare container configuration
	containerConfig := &container.Config{
		Image: "go-i2p-testnet",
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
		return fmt.Errorf("Error creating container: %v", err)
	}

	// Start the container
	StartOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, resp.ID, StartOptions); err != nil {
		return fmt.Errorf("Error starting container: %v", err)
	}

	return nil
}
func copyConfigToVolume(cli *client.Client, ctx context.Context, volumeName string, configData string) error {
	// Create a temporary container to copy data into the volume
	tempContainerConfig := &container.Config{
		Image:      "alpine",
		Tty:        false,
		WorkingDir: "/config",
		Cmd:        []string{"sh", "-c", "sleep 1d"},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/config", volumeName),
		},
	}

	resp, err := cli.ContainerCreate(ctx, tempContainerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("Error creating temporary container: %v", err)
	}
	defer func() {
		RemoveOptions := container.RemoveOptions{Force: true}
		cli.ContainerRemove(ctx, resp.ID, RemoveOptions)
	}()

	// Start the container
	StartOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, resp.ID, StartOptions); err != nil {
		return fmt.Errorf("Error starting temporary container: %v", err)
	}

	// Copy the configuration file into the container
	tarReader, err := createTarArchive("router.config", configData)
	if err != nil {
		return fmt.Errorf("Error creating tar archive: %v", err)
	}

	// Copy to the container's volume-mounted directory
	err = cli.CopyToContainer(ctx, resp.ID, "/config", tarReader, types.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("Error copying to container: %v", err)
	}

	// Stop the container
	StopOptions := container.StopOptions{}
	if err := cli.ContainerStop(ctx, resp.ID, StopOptions); err != nil {
		return fmt.Errorf("Error stopping temporary container: %v", err)
	}

	return nil
}

func createTarArchive(filename, content string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	hdr := &tar.Header{
		Name: filename,
		Mode: 0600,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf, nil
}

func buildDockerImage(cli *client.Client, ctx context.Context, imageName string, dockerfileDir string) error {
	dockerfileTar, err := archive.TarWithOptions(dockerfileDir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("Error creating tar archive of Dockerfile: %v", err)
	}

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
		Remove:     true,
	}

	resp, err := cli.ImageBuild(ctx, dockerfileTar, buildOptions)
	if err != nil {
		return fmt.Errorf("Error building Docker image: %v", err)
	}
	defer resp.Body.Close()

	// Read the output
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return fmt.Errorf("Error reading build output: %v", err)
	}
	fmt.Println("Docker build output:", buf.String())

	return nil
}
func imageExists(cli *client.Client, ctx context.Context, imageName string) (bool, error) {
	ListOptions := image.ListOptions{}
	images, err := cli.ImageList(ctx, ListOptions)
	if err != nil {
		return false, err
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == imageName || tag == imageName+":latest" {
				return true, nil
			}
		}
	}
	return false, nil
}
func main() {
	ctx := context.Background()

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error creating Docker client: %v", err)
	}
	// Check if the image exists
	exists, err := imageExists(cli, ctx, "go-i2p-testnet")
	if err != nil {
		log.Fatalf("Error checking for Docker image: %v", err)
	}

	if !exists {
		// Build the Docker image if it doesn't exist
		err = buildDockerImage(cli, ctx, "go-i2p-testnet", "../docker")
		if err != nil {
			log.Fatalf("Error building Docker image: %v", err)
		}
	} else {
		fmt.Printf("Docker image go-i2p-testnet already exists. Skipping build.\n")
	}
	/*
		// Build the Docker image
		err = buildDockerImage(cli, ctx, "go-i2p-testnet", "../docker")
		if err != nil {
			log.Fatalf("Error building Docker image: %v", err)
		}

	*/

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
		err := createRouterContainer(cli, ctx, routerID, ip, networkName, configData)
		if err != nil {
			log.Fatalf("Error creating router container: %v", err)
		}
		fmt.Printf("Started router%d with IP %s\n", routerID, ip)
	}
}
