package main

import (
	"context"
	"fmt"
	"github.com/chzyer/readline"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go-i2p-testnet/lib/docker_control"
	goi2p "go-i2p-testnet/lib/go-i2p"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

var (
	running = false
	// Track created containers and volumes for cleanup
	createdGOI2Prouters []string
	createdContainers   []string
	createdVolumes      []string
	mu                  sync.Mutex // To protect access to the slices
)

const (
	NETWORK = "go-i2p-testnet"
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

func addCreated(containerID, volumeID string) {
	mu.Lock()
	defer mu.Unlock()
	createdContainers = append(createdContainers, containerID)
	createdVolumes = append(createdVolumes, volumeID)
}

func start(cli *client.Client, ctx context.Context) {
	// Create Docker network
	networkName := NETWORK
	networkID, err := docker_control.CreateDockerNetwork(cli, ctx, networkName)
	if err != nil {
		log.Fatalf("Error creating Docker network: %v", err)
	}
	fmt.Printf("Created network %s with ID %s\n", networkName, networkID)

	//Create shared volume
	sharedVolumeName, err := docker_control.CreateSharedVolume(cli, ctx)
	if err != nil {
		log.Fatalf("error creating shared volume: %v", err)
	}
	createdVolumes = append(createdVolumes, sharedVolumeName)
	running = true
	fmt.Println("Network and shared volume created.")
	/*
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
			configData := goi2p.GenerateRouterConfig(routerID)

			// Create the container
			containerID, volumeName, err := goi2p.CreateRouterContainer(cli, ctx, routerID, ip, networkName, configData)
			if err != nil {
				log.Fatalf("Error creating router container: %v", err)
			}
			fmt.Printf("Started router%d with IP %s\n", routerID, ip)

			// Track the created container and volume for cleanup
			addCreated(containerID, volumeName)
		}

	*/

}

func addGOI2PRouter(cli *client.Client, ctx context.Context) error {
	mu.Lock()
	routerID := len(createdGOI2Prouters) + 1

	// Calculate next IP
	incr := routerID + 1
	if incr == 256 {
		return fmt.Errorf("maximum capacity reached (255)")
	}
	nextIP := fmt.Sprintf("172.28.0.%d", incr)
	mu.Unlock()

	configData := goi2p.GenerateRouterConfig(routerID)

	// Create the container
	containerID, volumeID, err := goi2p.CreateRouterContainer(cli, ctx, routerID, nextIP, NETWORK, configData)
	if err != nil {
		return err
	}

	mu.Lock()
	createdGOI2Prouters = append(createdGOI2Prouters, containerID)
	createdContainers = append(createdContainers, containerID)
	createdVolumes = append(createdVolumes, volumeID)

	addCreated(containerID, volumeID)

	fmt.Printf("Added router%d with IP %s\n", routerID, nextIP)

	return nil
}

func main() {
	ctx := context.Background()

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error creating Docker client: %v", err)
	}

	// Ensure cleanup is performed on exit
	defer func() {
		if running {
			cleanup(cli, ctx, createdContainers, createdVolumes, NETWORK)
		}
	}()

	// Set up signal handling to gracefully handle termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	//Begin command loop
	// Setup readline for command line input
	rl, err := readline.New("> ")
	if err != nil {
		log.Fatalf("Error initializing readline: %v", err)
	}
	defer rl.Close()
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
		case "rebuild":
			if running {
				fmt.Println("Testnet is running, not safe to rebuild")
			} else {
				err := docker_control.RebuildImages(cli, ctx)
				if err != nil {
					fmt.Printf("failed to rebuild images: %v\n", err)
				}
			}
		case "remove_images":
			if running {
				fmt.Println("Testnet is running, not safe to remove images")
			} else {
				err := docker_control.RemoveImages(cli, ctx)
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
	fmt.Println("  help					- Show this help message")
	fmt.Println("  start					- Start routers")
	fmt.Println("  stop					- Stop and cleanup routers")
	fmt.Println("  rebuild				- Rebuild docker images for nodes")
	fmt.Println("  remove_images			- Removes all node images")
	fmt.Println("  add_goi2p_router		- Add a router node (go-i2p)")
	fmt.Println("  exit					- Exit the CLI")
}
