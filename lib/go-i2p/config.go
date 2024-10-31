package go_i2p

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-i2p/go-i2p/lib/config"
	"go-i2p-testnet/lib/utils"
	"log"
	"os"
	"path/filepath"
)

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

func CopyConfigToVolume(cli *client.Client, ctx context.Context, volumeName string, configData string) error {
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
		return fmt.Errorf("error creating temporary container: %v", err)
	}
	defer func() {
		RemoveOptions := container.RemoveOptions{Force: true}
		err := cli.ContainerRemove(ctx, resp.ID, RemoveOptions)
		if err != nil {
			log.Printf("failed to remove container: %v", err)
		}
	}()

	// Start the container
	StartOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, resp.ID, StartOptions); err != nil {
		return fmt.Errorf("error starting temporary container: %v", err)
	}

	// Copy the configuration file into the container
	tarReader, err := utils.CreateTarArchive("router.config", configData)
	if err != nil {
		return fmt.Errorf("error creating tar archive: %v", err)
	}

	// Copy to the container's volume-mounted directory
	err = cli.CopyToContainer(ctx, resp.ID, "/config", tarReader, container.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("error copying to container: %v", err)
	}

	// Stop the container
	StopOptions := container.StopOptions{}
	if err := cli.ContainerStop(ctx, resp.ID, StopOptions); err != nil {
		return fmt.Errorf("error stopping temporary container: %v", err)
	}

	return nil
}

func GenerateRouterConfig(routerID int, ip string, peers []string) string {
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
