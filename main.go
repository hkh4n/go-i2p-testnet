package main

import (
	"context"
	"fmt"
	"github.com/chzyer/readline"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"go-i2p-testnet/lib/docker_control"
	goi2p "go-i2p-testnet/lib/go-i2p"
	"go-i2p-testnet/lib/i2pd"
	"go-i2p-testnet/lib/utils/logger"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

var (
	running = false
	// Track created containers and volumes for cleanup
	createdRouters    []string
	createdContainers []string
	createdVolumes    []string
	mu                sync.Mutex // To protect access to the slices
	log               = logger.GetTestnetLogger()
)

const (
	NETWORK = "go-i2p-testnet"
)

// cleanup removes all created Docker resources: containers, volumes, and network.
func cleanup(cli *client.Client, ctx context.Context, createdContainers []string, createdVolumes []string, networkName string) {
	log.WithField("networkName", networkName).Debug("Starting cleanup of Docker resources")

	// Remove containers
	for _, containerID := range createdContainers {
		log.WithField("containerID", containerID).Debug("Attempting to stop container")
		// Attempt to stop the container
		timeout := 10 // seconds
		err := cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
		if err != nil {
			log.WithFields(map[string]interface{}{
				"containerID": containerID,
				"error":       err,
			}).Error("Failed to stop container")
		}

		// Attempt to remove the container
		log.WithField("containerID", containerID).Debug("Attempting to remove container")
		removeOptions := container.RemoveOptions{Force: true}
		err = cli.ContainerRemove(ctx, containerID, removeOptions)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"containerID": containerID,
				"error":       err,
			}).Error("Failed to remove container")
		} else {
			log.WithField("containerID", containerID).Debug("Successfully removed container")
		}
	}

	// Remove volumes
	for _, volumeName := range createdVolumes {
		log.WithField("volumeName", volumeName).Debug("Attempting to remove volume")
		err := cli.VolumeRemove(ctx, volumeName, true)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"volumeName": volumeName,
				"error":      err,
			}).Error("Failed to remove volume")
		} else {
			log.WithField("volumeName", volumeName).Debug("Successfully removed volume")
		}
	}

	// Remove network
	log.WithField("networkName", networkName).Debug("Attempting to remove network")
	err := cli.NetworkRemove(ctx, networkName)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"networkName": networkName,
			"error":       err,
		}).Error("Failed to remove network")
	} else {
		log.WithField("networkName", networkName).Debug("Successfully removed network")
	}
}

func addCreated(containerID, volumeID string) {
	//mu.Lock() //For some reason, this freezes
	//defer mu.Unlock()
	log.WithFields(map[string]interface{}{
		"containerID": containerID,
		"volumeID":    volumeID,
	}).Debug("Tracking new container and volume")
	createdContainers = append(createdContainers, containerID)
	createdVolumes = append(createdVolumes, volumeID)
	return
}

func start(cli *client.Client, ctx context.Context) {
	log.Debug("Starting testnet initialization")
	// Create Docker network
	networkName := NETWORK
	log.WithField("networkName", networkName).Debug("Creating Docker network")
	networkID, err := docker_control.CreateDockerNetwork(cli, ctx, networkName)
	if err != nil {
		log.WithError(err).Fatal("Failed to create Docker network")
		//log.Fatalf("Error creating Docker network: %v", err)
	}
	log.WithFields(map[string]interface{}{
		"networkName": networkName,
		"networkID":   networkID,
	}).Debug("Successfully created network")

	//Create shared volume
	log.Debug("Creating shared volume")
	sharedVolumeName, err := docker_control.CreateSharedVolume(cli, ctx)
	if err != nil {
		log.Fatalf("error creating shared volume: %v", err)
	}
	createdVolumes = append(createdVolumes, sharedVolumeName)
	running = true
	log.WithField("volumeName", sharedVolumeName).Debug("Successfully created shared volume")
}

func status(cli *client.Client, ctx context.Context) {
	log.Debug("Fetching status of router containers")

	// List all containers (both running and stopped)
	containerListOptions := container.ListOptions{
		All: true,
	}
	containers, err := cli.ContainerList(ctx, containerListOptions)
	if err != nil {
		log.WithError(err).Error("Failed to list Docker containers")
		fmt.Println("Error: failed to list Docker containers:", err)
		return
	}

	// Filter containers whose names start with "router"
	fmt.Println("Current router containers:")
	found := false
	for _, container := range containers {
		for _, name := range container.Names {
			// Docker prepends "/" to container names
			if strings.HasPrefix(name, "/router") {
				found = true
				fmt.Printf("Container ID: %s, Name: %s, Image: %s, Status: %s\n",
					container.ID[:12], name[1:], container.Image, container.Status)
			}
		}
	}
	if !found {
		fmt.Println("No router containers are running.")
	}
}
func addGOI2PRouter(cli *client.Client, ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()
	routerID := len(createdRouters) + 1

	log.WithField("routerID", routerID).Debug("Adding new go-i2p router")

	// Calculate next IP
	incr := routerID + 1
	if incr == 256 {
		log.Error("Maximum number of nodes reached (255)")
		return fmt.Errorf("too many nodes! (255)")
	}
	nextIP := fmt.Sprintf("172.28.0.%d", incr)

	log.WithFields(map[string]interface{}{
		"routerID": routerID,
		"ip":       nextIP,
	}).Debug("Generating router configuration")

	configData := goi2p.GenerateRouterConfig(routerID)

	// Create the container
	log.Debug("Creating router container")
	containerID, volumeID, err := goi2p.CreateRouterContainer(cli, ctx, routerID, nextIP, NETWORK, configData)
	if err != nil {
		log.WithError(err).Error("Failed to create router container")
		return err
	}

	log.WithFields(map[string]interface{}{
		"routerID":    routerID,
		"containerID": containerID,
		"volumeID":    volumeID,
		"ip":          nextIP,
	}).Debug("Adding router to tracking lists")
	createdRouters = append(createdRouters, containerID)
	createdContainers = append(createdContainers, containerID)
	createdVolumes = append(createdVolumes, volumeID)

	addCreated(containerID, volumeID)
	return nil
}

func addI2PDRouter(cli *client.Client, ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()
	routerID := len(createdRouters) + 1

	log.WithField("routerID", routerID).Debug("Adding new i2pd router")

	// Calculate next IP
	incr := routerID + 1
	if incr == 256 {
		log.Error("Maximum number of nodes reached (255)")
		return fmt.Errorf("too many nodes! (255)")
	}
	nextIP := fmt.Sprintf("172.28.0.%d", incr)

	log.WithFields(map[string]interface{}{
		"routerID": routerID,
		"ip":       nextIP,
	}).Debug("Generating router configuration")

	// Generate the configuration data
	configData, err := i2pd.GenerateDefaultRouterConfig(routerID)
	if err != nil {
		log.WithError(err).Error("Failed to generate i2pd router config")
		return err
	}

	// Create configuration volume
	volumeName := fmt.Sprintf("i2pd_router%d_config", routerID)
	createOptions := volume.CreateOptions{
		Name: volumeName,
	}
	_, err = cli.VolumeCreate(ctx, createOptions)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"volumeName": volumeName,
			"error":      err,
		}).Error("Failed to create volume")
		return err
	}

	// Copy configuration to volume
	err = i2pd.CopyConfigToVolume(cli, ctx, volumeName, configData)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"volumeName": volumeName,
			"error":      err,
		}).Error("Failed to copy config to volume")
		return err
	}

	// Create and start router container
	containerID, err := i2pd.CreateRouterContainer(cli, ctx, routerID, nextIP, NETWORK, volumeName)
	if err != nil {
		log.WithError(err).Error("Failed to create i2pd router container")
		return err
	}

	log.WithFields(map[string]interface{}{
		"routerID":    routerID,
		"containerID": containerID,
		"volumeID":    volumeName,
		"ip":          nextIP,
	}).Debug("Adding router to tracking lists")

	// Update tracking lists
	createdRouters = append(createdRouters, containerID)
	createdContainers = append(createdContainers, containerID)
	createdVolumes = append(createdVolumes, volumeName)

	// Add to any additional tracking structures if necessary
	addCreated(containerID, volumeName)
	return nil
}

func main() {
	ctx := context.Background()

	// Initialize Docker client
	log.Debug("Initializing Docker client")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.WithError(err).Fatal("Failed to create Docker client")
	}

	// Ensure cleanup is performed on exit
	defer func() {
		if running {
			log.Debug("Performing cleanup on exit")
			cleanup(cli, ctx, createdContainers, createdVolumes, NETWORK)
		}
	}()

	// Set up signal handling to gracefully handle termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	//Begin command loop
	// Setup readline for command line input
	log.Debug("Initializing readline interface")
	rl, err := readline.New("> ")
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize readline")
	}
	defer rl.Close()
	log.Debug("Starting command loop")
	fmt.Println("Logging is available, check README.md for details. Set env DEBUG_TESTNET to debug, warn or error")
	for {
		line, err := rl.Readline()
		if err != nil { //EOF
			break
		}
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		log.WithField("command", parts[0]).Debug("Processing command")

		// handle commands
		switch parts[0] {
		case "help":
			showHelp()
		case "start":
			if running {
				fmt.Println("Testnet is already running")
			} else {
				start(cli, ctx)
			}
		case "stop":
			if running {
				cleanup(cli, ctx, createdContainers, createdVolumes, NETWORK)
				running = false
			} else {
				fmt.Println("Testnet isn't running")
			}
		case "status":
			status(cli, ctx)
		case "build":
			if running {
				fmt.Println("Testnet is running, not safe to build")
			} else {
				err := buildImages(cli, ctx)
				if err != nil {
					fmt.Printf("failed to build images: %v\n", err)
				}
			}
		case "rebuild":
			if running {
				fmt.Println("Testnet is running, not safe to rebuild")
			} else {
				err := rebuildImages(cli, ctx)
				if err != nil {
					fmt.Printf("failed to rebuild images: %v\n", err)
				}
			}
		case "remove_images":
			if running {
				fmt.Println("Testnet is running, not safe to remove images")
			} else {
				err := removeImages(cli, ctx)
				if err != nil {
					fmt.Printf("failed to remove images: %v\n", err)
				}
			}
		case "add_goi2p_router":
			if !running {
				fmt.Println("Testnet isn't running")
			} else {
				err := addGOI2PRouter(cli, ctx)
				if err != nil {
					fmt.Printf("failed to add router: %v\n", err)
				}
			}
		case "add_i2pd_router":
			if !running {
				fmt.Println("Testnet isn't running")
			} else {
				err := addI2PDRouter(cli, ctx)
				if err != nil {
					fmt.Printf("failed to add router: %v\n", err)
				}
			}
		case "exit":
			fmt.Println("Exiting...")
			if running {
				cleanup(cli, ctx, createdContainers, createdVolumes, NETWORK)
			}
			return
		default:
			fmt.Println("Unknown command. Type 'help' for a list of commands")
		}
	}

	// Wait for interrupt signal to gracefully shutdown
	<-sigs
	fmt.Println("\nReceived interrupt signal. Initiating cleanup...")
}

func showHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help						- Show this help message")
	fmt.Println("  start						- Start the testnet")
	fmt.Println("  stop						- Stop testnet and cleanup routers")
	fmt.Println("  status 						- Show status")
	fmt.Println("  build						- Build docker images for nodes")
	fmt.Println("  rebuild					- Rebuild docker images for nodes")
	fmt.Println("  remove_images					- Removes all node images")
	fmt.Println("  add_goi2p_router				- Add a router node (go-i2p)")
	fmt.Println("  add_i2pd_router				- Add a router node (i2pd)")
	fmt.Println("  exit						- Exit the CLI")
}

func buildImages(cli *client.Client, ctx context.Context) error {
	log.Debug("Building go-i2p node image")
	err := goi2p.BuildImage(cli, ctx)
	if err != nil {
		log.WithError(err).Error("Failed to build go-i2p node image")
		return err
	}
	log.Debug("Successfully built go-i2p node image")

	log.Debug("Building i2pd node image")
	err = i2pd.BuildImage(cli, ctx)
	if err != nil {
		log.WithError(err).Error("Failed to build i2pd node image")
		return err
	}
	log.Debug("Successfully built i2pd node image")

	return nil
}

func removeImages(cli *client.Client, ctx context.Context) error {
	log.Debug("Removing go-i2p node image")
	err := goi2p.RemoveImage(cli, ctx)
	if err != nil {
		log.WithError(err).Error("Failed to remove go-i2p node image")
		return err
	}
	log.Debug("Successfully removed go-i2p node image")

	log.Debug("Removing i2pd node image")
	err = i2pd.RemoveImage(cli, ctx)
	if err != nil {
		log.WithError(err).Error("Failed to remove i2pd node image")
		return err
	}
	log.Debug("Successfully removed i2pd node image")

	return nil
}

func rebuildImages(cli *client.Client, ctx context.Context) error {
	log.Debug("Starting image rebuild process")
	err := removeImages(cli, ctx)
	if err != nil {
		log.WithError(err).Error("Failed during image removal step of rebuild")
		return err
	}
	err = buildImages(cli, ctx)
	if err != nil {
		log.WithError(err).Error("Failed during image build step of rebuild")
		return err
	}

	log.Debug("Successfully completed image rebuild")
	return nil
}
