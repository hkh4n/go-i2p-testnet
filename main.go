package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chzyer/readline"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/go-i2p/go-i2p/lib/common/base64"
	"github.com/go-i2p/go-i2p/lib/common/router_info"
	"go-i2p-testnet/lib/docker_control"
	goi2pnode "go-i2p-testnet/lib/go-i2p"
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
	sharedVolumeName  string
	mu                sync.Mutex // To protect access to the slices
	log               = logger.GetTestnetLogger()
)

var completer = readline.NewPrefixCompleter(
	readline.PcItem("help"),
	readline.PcItem("start"),
	readline.PcItem("stop"),
	readline.PcItem("status"),
	readline.PcItem("usage"),
	readline.PcItem("build"),
	readline.PcItem("rebuild"),
	readline.PcItem("remove_images"),
	readline.PcItem("add",
		readline.PcItem("goi2p_router"),
		readline.PcItem("i2pd_router"),
	),
	readline.PcItem("sync_i2pd_shared"),
	readline.PcItem("sync-i2pd_netdb"),
	readline.PcItem("exit"),
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
	}
	log.WithFields(map[string]interface{}{
		"networkName": networkName,
		"networkID":   networkID,
	}).Debug("Successfully created network")

	//Create shared volume
	log.Debug("Creating shared volume")
	sharedVolumeName, err = docker_control.CreateSharedVolume(cli, ctx)
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
	for _, _container := range containers {
		for _, name := range _container.Names {
			// Docker prepends "/" to _container names
			if strings.HasPrefix(name, "/router") {
				found = true
				fmt.Printf("Container ID: %s, Name: %s, Image: %s, Status: %s\n",
					_container.ID[:12], name[1:], _container.Image, _container.Status)
			}
		}
	}
	if !found {
		fmt.Println("No router containers are running.")
	}
}
func usage(cli *client.Client, ctx context.Context) {
	log.Debug("Fetching usage statistics for router containers")

	// List all containers
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		log.WithError(err).Error("Failed to list Docker containers")
		fmt.Println("Error: failed to list Docker containers:", err)
		return
	}

	found := false
	fmt.Println("\nRouter Container Usage Statistics:")
	fmt.Printf("%-20s %-20s %-20s %-10s\n", "NAME", "MEMORY USAGE (MB)", "MEMORY LIMIT (MB)", "CPU %")
	fmt.Println(strings.Repeat("-", 75))

	for _, c := range containers {
		// Filter for router containers
		for _, name := range c.Names {
			if strings.HasPrefix(name, "/router") {
				found = true

				// Get container stats
				stats, err := cli.ContainerStats(ctx, c.ID, false)
				if err != nil {
					log.WithFields(map[string]interface{}{
						"containerID": c.ID,
						"error":       err,
					}).Error("Failed to get container stats")
					continue
				}

				// Decode stats
				var v *container.StatsResponse
				decoder := json.NewDecoder(stats.Body)
				err = decoder.Decode(&v)
				stats.Body.Close()

				if err != nil {
					log.WithError(err).Error("Failed to decode container stats")
					continue
				}

				// Calculate memory usage in MB
				memUsageMB := float64(v.MemoryStats.Usage) / 1024 / 1024 // Convert to MB
				memLimitMB := float64(v.MemoryStats.Limit) / 1024 / 1024 // Convert to MB

				// Calculate CPU percentage
				cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
				systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
				cpuPercent := 0.0
				if systemDelta > 0 && cpuDelta > 0 {
					cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100
				}

				fmt.Printf("%-20s %-20.2f %-20.2f %-10.2f\n",
					strings.TrimPrefix(name, "/"),
					memUsageMB,
					memLimitMB,
					cpuPercent)
			}
		}
	}

	if !found {
		fmt.Println("No router containers found.")
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

	configData := goi2pnode.GenerateRouterConfig(routerID)

	// Create the container
	log.Debug("Creating router container")
	containerID, volumeID, err := goi2pnode.CreateRouterContainer(cli, ctx, routerID, nextIP, NETWORK, configData)
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
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[31mgo-i2p-testnet»\033[0m ",
		AutoComplete:    completer,
		HistoryFile:     "/tmp/readline.tmp", // Optional: Enable command history
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
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
		case "usage":
			if !running {
				fmt.Println("Testnet isn't running")
			} else {
				usage(cli, ctx)
			}
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
		case "add":
			if len(parts) < 2 {
				fmt.Println("Specify the type of router to add. Usage: add [goi2p_router|i2pd_router]")
				continue
			}
			switch parts[1] {
			case "goi2p_router":
				if !running {
					fmt.Println("Testnet isn't running")
				} else {
					err := addGOI2PRouter(cli, ctx)
					if err != nil {
						fmt.Printf("failed to add router: %v\n", err)
					}
				}
			case "i2pd_router":
				if !running {
					fmt.Println("Testnet isn't running")
				} else {
					err := addI2PDRouter(cli, ctx)
					if err != nil {
						fmt.Printf("failed to add router: %v\n", err)
					}
				}
			default:
				fmt.Println("Unknown router type. Available types: goi2p_router, i2pd_router")
			}
		case "sync_i2pd_shared":
			if !running {
				fmt.Println("Testnet isn't running")
			} else {
				log.Debug("Syncing netDb from all router containers to the shared volume")

				// Iterate through all created router containers
				for _, containerID := range createdContainers {
					log.WithField("containerID", containerID).Debug("Syncing netDb for container")

					// Sync the netDb directory to the shared volume
					err := i2pd.SyncNetDbToShared(cli, ctx, containerID, sharedVolumeName) // Pass sharedVolumeName
					if err != nil {
						fmt.Printf("Failed to sync netDb from container %s: %v\n", containerID, err)
					} else {
						fmt.Printf("Successfully synced netDb from container %s to shared volume\n", containerID)
					}
				}
			}
		case "sync_i2pd_netdb":
			if !running {
				fmt.Println("Testnet isn't running")
			} else {
				log.Debug("Syncing netDb from shared volume to all router containers")

				// Sync from shared volume to all router containers
				for _, containerID := range createdContainers {
					log.WithField("containerID", containerID).Debug("Syncing netDb from shared volume to container")

					// Sync the shared netDb to the container
					err := i2pd.SyncSharedToNetDb(cli, ctx, containerID, sharedVolumeName)
					if err != nil {
						fmt.Printf("Failed to sync netDb to container %s: %v\n", containerID, err)
						continue
					} else {
						fmt.Printf("Successfully synced netDb to container %s from shared volume\n", containerID)
					}
				}

				// Sync each container's RouterInfo back to the shared netDb
				log.Debug("Syncing RouterInfo from each container to the shared netDb")
				for _, containerID := range createdContainers {
					log.WithField("containerID", containerID).Debug("Syncing RouterInfo from container to shared netDb")

					// Sync the RouterInfo from the container to the shared netDb
					err := i2pd.SyncRouterInfoToNetDb(cli, ctx, containerID, sharedVolumeName)
					if err != nil {
						fmt.Printf("Failed to sync RouterInfo from container %s to shared netDb: %v\n", containerID, err)
					} else {
						fmt.Printf("Successfully synced RouterInfo from container %s to shared netDb\n", containerID)
					}
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

		case "hidden": // This is used for debugging and experimental reasons, not meant to be used for the end user
			if len(parts) < 2 {
				fmt.Println("Specify hidden command")
				continue
			}
			switch parts[1] {
			case "extract":
				if len(parts) < 4 {
					fmt.Println("Usage: hidden extract <containerID> <filePath>")
					continue
				}

				containerID := parts[2]
				filePath := parts[3]

				content, err := docker_control.ReadFileFromContainer(cli, ctx, containerID, filePath)
				if err != nil {
					fmt.Printf("Error extracting file: %v\n", err)
					continue
				}

				fmt.Println("File content:")
				fmt.Println(content)
			case "read_router_info":
				if len(parts) < 4 {
					fmt.Println("Usage: hidden read_router_info <containerID> <filePath>")
					continue
				}

				containerID := parts[2]
				filePath := parts[3]

				content, err := docker_control.ReadFileFromContainer(cli, ctx, containerID, filePath)
				if err != nil {
					fmt.Printf("Error extracting file: %v\n", err)
					continue
				}

				ri, _, err := router_info.ReadRouterInfo([]byte(content))
				if err != nil {
					fmt.Printf("Error reading router info: %v\n", err)
					continue
				}
				fmt.Println("Successfully read router info")
				fmt.Printf("Options: %v\n", ri.Options())
				fmt.Printf("Signature: %s\n", ri.Signature())
				fmt.Printf("GoodVersion: %v\n", ri.GoodVersion())
				identHash := ri.IdentHash()
				encodedHash := base64.EncodeToString(identHash[:])
				fmt.Printf("IdentHash: %v\n", encodedHash)
				fmt.Printf("Network: %v\n", ri.Network())
				fmt.Printf("Peersize: %v\n", ri.PeerSize())
				fmt.Printf("Published: %v\n", ri.Published())
				fmt.Printf("Reachable: %v\n", ri.Reachable())
				fmt.Printf("RouterAddressCount: %v\n", ri.RouterAddressCount())
				fmt.Printf("RouterAddresses: %v\n", ri.RouterAddresses())
				fmt.Printf("RouterIdentity: %v\n", ri.RouterIdentity())
				fmt.Printf("RouterVersion: %v\n", ri.RouterVersion())
				fmt.Printf("UnCongested: %v\n", ri.UnCongested())
			}
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
	fmt.Println("  status					- Show status")
	fmt.Println("  usage                  			    - Show memory and CPU usage of router containers")
	fmt.Println("  build						- Build docker images for nodes")
	fmt.Println("  rebuild					- Rebuild docker images for nodes")
	fmt.Println("  remove_images					- Removes all node images")
	fmt.Println("  add <nodetype> 				- Available node types are go-i2p and i2pd")
	fmt.Println("  exit						- Exit the CLI")
}

func buildImages(cli *client.Client, ctx context.Context) error {
	log.Debug("Building go-i2p node image")
	err := goi2pnode.BuildImage(cli, ctx)
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
	err := goi2pnode.RemoveImage(cli, ctx)
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
