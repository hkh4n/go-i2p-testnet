package go_i2p

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-i2p/go-i2p/lib/config"
	"go-i2p-testnet/lib/utils"
	"go-i2p-testnet/lib/utils/logger"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var log = logger.GetTestnetLogger()

// initializeRouterConfig sets up a router-specific configuration for each instance
func initializeRouterConfig(routerID int) *config.RouterConfig {
	log.WithField("routerID", routerID).Debug("Initializing router configuration")
	// Define base directory for this router's configuration
	baseDir := filepath.Join("testnet", fmt.Sprintf("router%d", routerID))
	err := os.MkdirAll(baseDir, os.ModePerm)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"routerID": routerID,
			"baseDir":  baseDir,
			"error":    err,
		}).Error("Failed to create base directory")
		return nil
	}

	// Assign each router its own netDb and working directory
	netDbPath := filepath.Join(baseDir, "netDb")
	workingDir := filepath.Join(baseDir, "config")

	log.WithFields(map[string]interface{}{
		"routerID":   routerID,
		"netDbPath":  netDbPath,
		"workingDir": workingDir,
	}).Debug("Creating router directories")

	err = os.MkdirAll(netDbPath, os.ModePerm)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"routerID":  routerID,
			"netDbPath": netDbPath,
			"error":     err,
		}).Error("Failed to create netDb directory")
		return nil
	}
	err = os.MkdirAll(workingDir, os.ModePerm)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"routerID":   routerID,
			"workingDir": workingDir,
			"error":      err,
		}).Error("Failed to create working directory")
		return nil
	}

	log.WithFields(map[string]interface{}{
		"routerID":   routerID,
		"baseDir":    baseDir,
		"workingDir": workingDir,
		"netDbPath":  netDbPath,
	}).Debug("Router configuration initialized successfully")

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
	log.WithField("volumeName", volumeName).Debug("Starting config copy to volume")

	tempContainerConfig := &container.Config{
		Image:      "alpine",
		Tty:        false,
		WorkingDir: "/config",
		Cmd:        []string{"sh", "-c", "mkdir -p /config/.go-i2p && sleep 1d"},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/config", volumeName),
		},
	}

	log.WithFields(map[string]interface{}{
		"image":      tempContainerConfig.Image,
		"volumeName": volumeName,
	}).Debug("Creating temporary container")

	resp, err := cli.ContainerCreate(ctx, tempContainerConfig, hostConfig, nil, nil, "")
	if err != nil {
		log.WithError(err).Error("Failed to create temporary container")
		return fmt.Errorf("error creating temporary container: %v", err)
	}
	defer func() {
		log.WithField("containerID", resp.ID).Debug("Removing temporary container")
		RemoveOptions := container.RemoveOptions{Force: true}
		err := cli.ContainerRemove(ctx, resp.ID, RemoveOptions)
		if err != nil {
			log.WithError(err).Error("Failed to remove temporary container")
		}
	}()

	// Start the container
	log.WithField("containerID", resp.ID).Debug("Starting temporary container")
	StartOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, resp.ID, StartOptions); err != nil {
		log.WithError(err).Error("Failed to start temporary container")
		return fmt.Errorf("error starting temporary container: %v", err)
	}

	// Copy the configuration file into the container
	log.Debug("Creating tar archive of config data")
	tarReader, err := utils.CreateTarArchive(".go-i2p/config.yaml", configData)
	if err != nil {
		log.WithError(err).Error("Failed to create tar archive")
		return fmt.Errorf("error creating tar archive: %v", err)
	}

	// Copy to the container's volume-mounted directory
	log.WithField("containerID", resp.ID).Debug("Copying config to container")
	err = cli.CopyToContainer(ctx, resp.ID, "/config", tarReader, container.CopyToContainerOptions{})
	if err != nil {
		log.WithError(err).Error("Failed to copy config to container")
		return fmt.Errorf("error copying to container: %v", err)
	}

	// Stop the container
	log.WithField("containerID", resp.ID).Debug("Stopping temporary container")
	StopOptions := container.StopOptions{}
	if err := cli.ContainerStop(ctx, resp.ID, StopOptions); err != nil {
		log.WithError(err).Error("Failed to stop temporary container")
		return fmt.Errorf("error stopping temporary container: %v", err)
	}

	log.Debug("Successfully copied config to volume")
	return nil
}

func GenerateRouterConfig(routerID int) string {
	log.WithField("routerID", routerID).Debug("Starting router config generation")
	// Initialize router-specific configuration
	routerConfig := initializeRouterConfig(routerID)
	log.WithField("routerID", routerID).Error("Failed to initialize router config")
	// Define common settings for each router instance
	log.Debug("Marshaling router configuration to YAML")
	configDataYAML, err := yaml.Marshal(routerConfig)
	if err != nil {
		log.WithError(err).Error("Failed to marshal router configuration")
		panic(err)
	}
	configDataYAMLstr := string(configDataYAML)
	log.WithFields(map[string]interface{}{
		"routerID": routerID,
		"config":   configDataYAMLstr,
	}).Debug("Router configuration generated successfully")
	return configDataYAMLstr
}
