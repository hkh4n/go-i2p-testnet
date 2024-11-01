package i2pd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go-i2p-testnet/lib/utils"
	"gopkg.in/ini.v1"
)

type I2PDConfig struct {
	// Global options (before any section)
	TunnelsConf string `ini:"tunconf"`
	TunnelsDir  string `ini:"tunnelsdir"`
	CertsDir    string `ini:"certsdir"`
	Pidfile     string `ini:"pidfile"`
	Log         string `ini:"log"`
	Logfile     string `ini:"logfile"`
	Loglevel    string `ini:"loglevel"`
	Logclftime  bool   `ini:"logclftime"`
	Daemon      bool   `ini:"daemon"`
	Family      string `ini:"family"`
	Ifname      string `ini:"ifname"`
	Ifname4     string `ini:"ifname4"`
	Ifname6     string `ini:"ifname6"`
	Address4    string `ini:"address4"`
	Address6    string `ini:"address6"`
	Host        string `ini:"host"`
	Port        int    `ini:"port"`
	IPv4        bool   `ini:"ipv4"`
	IPv6        bool   `ini:"ipv6"`
	SSU         bool   `ini:"ssu"`
	Bandwidth   string `ini:"bandwidth"`
	Share       int    `ini:"share"`
	Notransit   bool   `ini:"notransit"`
	Floodfill   bool   `ini:"floodfill"`

	// Sections
	NTCP2          NTCP2Config          `ini:"ntcp2"`
	SSU2           SSU2Config           `ini:"ssu2"`
	HTTP           HTTPConfig           `ini:"http"`
	HTTPProxy      HTTPProxyConfig      `ini:"httpproxy"`
	SocksProxy     SocksProxyConfig     `ini:"socksproxy"`
	SAM            SAMConfig            `ini:"sam"`
	BOB            BOBConfig            `ini:"bob"`
	I2CP           I2CPConfig           `ini:"i2cp"`
	I2PControl     I2PControlConfig     `ini:"i2pcontrol"`
	Precomputation PrecomputationConfig `ini:"precomputation"`
	UPnP           UPnPConfig           `ini:"upnp"`
	Meshnets       MeshnetsConfig       `ini:"meshnets"`
	Reseed         ReseedConfig         `ini:"reseed"`
	Addressbook    AddressbookConfig    `ini:"addressbook"`
	Limits         LimitsConfig         `ini:"limits"`
	Trust          TrustConfig          `ini:"trust"`
	Exploratory    ExploratoryConfig    `ini:"exploratory"`
	Persist        PersistConfig        `ini:"persist"`
	CPUExt         CPUExtConfig         `ini:"cpuext"`
}

// NTCP2 Section
type NTCP2Config struct {
	Enabled   bool `ini:"enabled"`
	Published bool `ini:"published"`
	Port      int  `ini:"port"`
}

// SSU2 Section
type SSU2Config struct {
	Enabled   bool `ini:"enabled"`
	Published bool `ini:"published"`
	Port      int  `ini:"port"`
}

// HTTP Section
type HTTPConfig struct {
	Enabled bool   `ini:"enabled"`
	Address string `ini:"address"`
	Port    int    `ini:"port"`
	Webroot string `ini:"webroot"`
	Auth    bool   `ini:"auth"`
	User    string `ini:"user"`
	Pass    string `ini:"pass"`
	Lang    string `ini:"lang"`
}

// HTTPProxy Section
type HTTPProxyConfig struct {
	Enabled       bool   `ini:"enabled"`
	Address       string `ini:"address"`
	Port          int    `ini:"port"`
	Keys          string `ini:"keys"`
	AddressHelper bool   `ini:"addresshelper"`
	Outproxy      string `ini:"outproxy"`
}

// SocksProxy Section
type SocksProxyConfig struct {
	Enabled         bool   `ini:"enabled"`
	Address         string `ini:"address"`
	Port            int    `ini:"port"`
	Keys            string `ini:"keys"`
	OutproxyEnabled bool   `ini:"outproxy.enabled"`
	Outproxy        string `ini:"outproxy"`
	OutproxyPort    int    `ini:"outproxyport"`
}

// SAM Section
type SAMConfig struct {
	Enabled bool   `ini:"enabled"`
	Address string `ini:"address"`
	Port    int    `ini:"port"`
}

// BOB Section
type BOBConfig struct {
	Enabled bool   `ini:"enabled"`
	Address string `ini:"address"`
	Port    int    `ini:"port"`
}

// I2CP Section
type I2CPConfig struct {
	Enabled bool   `ini:"enabled"`
	Address string `ini:"address"`
	Port    int    `ini:"port"`
}

// I2PControl Section
type I2PControlConfig struct {
	Enabled  bool   `ini:"enabled"`
	Address  string `ini:"address"`
	Port     int    `ini:"port"`
	Password string `ini:"password"`
}

// Precomputation Section
type PrecomputationConfig struct {
	ElGamal bool `ini:"elgamal"`
}

// UPnP Section
type UPnPConfig struct {
	Enabled bool   `ini:"enabled"`
	Name    string `ini:"name"`
}

// Meshnets Section
type MeshnetsConfig struct {
	Yggdrasil  bool   `ini:"yggdrasil"`
	YggAddress string `ini:"yggaddress"`
}

// Reseed Section
type ReseedConfig struct {
	Verify    bool   `ini:"verify"`
	URLs      string `ini:"urls"`
	YggURLs   string `ini:"yggurls"`
	File      string `ini:"file"`
	ZipFile   string `ini:"zipfile"`
	Proxy     string `ini:"proxy"`
	Threshold int    `ini:"threshold"`
}

// Addressbook Section
type AddressbookConfig struct {
	DefaultURL    string `ini:"defaulturl"`
	Subscriptions string `ini:"subscriptions"`
}

// Limits Section
type LimitsConfig struct {
	TransitTunnels int `ini:"transittunnels"`
	OpenFiles      int `ini:"openfiles"`
	CoreSize       int `ini:"coresize"`
}

// Trust Section
type TrustConfig struct {
	Enabled bool   `ini:"enabled"`
	Family  string `ini:"family"`
	Routers string `ini:"routers"`
	Hidden  bool   `ini:"hidden"`
}

// Exploratory Section
type ExploratoryConfig struct {
	Inbound  TunnelConfig `ini:"inbound"`
	Outbound TunnelConfig `ini:"outbound"`
}

type TunnelConfig struct {
	Length   int `ini:"length"`
	Quantity int `ini:"quantity"`
}

// Persist Section
type PersistConfig struct {
	Profiles    bool `ini:"profiles"`
	Addressbook bool `ini:"addressbook"`
}

// CPU Extensions Section
type CPUExtConfig struct {
	AESNI bool `ini:"aesni"`
	AVX   bool `ini:"avx"`
	Force bool `ini:"force"`
}

func GenerateDefaultI2PDConfig() *I2PDConfig {
	return &I2PDConfig{
		// Global options (before any section)
		TunnelsConf: "",
		TunnelsDir:  "",
		CertsDir:    "",
		Pidfile:     "",
		Log:         "stdout",
		Logfile:     "",
		Loglevel:    "warn",
		Logclftime:  false,
		Daemon:      false,
		Family:      "",
		Ifname:      "",
		Ifname4:     "",
		Ifname6:     "",
		Address4:    "",
		Address6:    "",
		Host:        "",
		Port:        0, // Default is a random port
		IPv4:        true,
		IPv6:        true,
		SSU:         true,
		Bandwidth:   "L",
		Share:       100,
		Notransit:   false,
		Floodfill:   false,

		// NTCP2 section
		NTCP2: NTCP2Config{
			Enabled:   true,
			Published: true,
			Port:      0, // Uses global port option
		},

		// SSU2 section
		SSU2: SSU2Config{
			Enabled:   true,
			Published: true,
			Port:      0, // Uses global port option or port + 1 if SSU is enabled
		},

		// HTTP section
		HTTP: HTTPConfig{
			Enabled: true,
			Address: "127.0.0.1",
			Port:    7070,
			Webroot: "/",
			Auth:    false,
			User:    "",
			Pass:    "",
			Lang:    "english",
		},

		// HTTP Proxy section
		HTTPProxy: HTTPProxyConfig{
			Enabled:       true,
			Address:       "127.0.0.1",
			Port:          4444,
			Keys:          "",
			AddressHelper: true,
			Outproxy:      "",
		},

		// SOCKS Proxy section
		SocksProxy: SocksProxyConfig{
			Enabled:         true,
			Address:         "127.0.0.1",
			Port:            4447,
			Keys:            "",
			OutproxyEnabled: false,
			Outproxy:        "",
			OutproxyPort:    0,
		},

		// SAM section
		SAM: SAMConfig{
			Enabled: true,
			Address: "127.0.0.1",
			Port:    7656,
		},

		// BOB section
		BOB: BOBConfig{
			Enabled: false,
			Address: "127.0.0.1",
			Port:    2827,
		},

		// I2CP section
		I2CP: I2CPConfig{
			Enabled: false,
			Address: "127.0.0.1",
			Port:    7654,
		},

		// I2PControl section
		I2PControl: I2PControlConfig{
			Enabled:  false,
			Address:  "127.0.0.1",
			Port:     7650,
			Password: "itoopie",
		},

		// Precomputation section
		Precomputation: PrecomputationConfig{
			ElGamal: true, // Enabled by default
		},

		// UPnP section
		UPnP: UPnPConfig{
			Enabled: false, // Disabled by default on non-Windows/Android platforms
			Name:    "I2Pd",
		},

		// Meshnets section
		Meshnets: MeshnetsConfig{
			Yggdrasil:  false,
			YggAddress: "",
		},

		// Reseed section
		Reseed: ReseedConfig{
			Verify: true,
			URLs: "https://reseed.i2p-projekt.de/," +
				"https://i2p.mooo.com/netDb/," +
				"https://netdb.i2p2.no/",
			YggURLs:   "",
			File:      "",
			ZipFile:   "",
			Proxy:     "",
			Threshold: 25,
		},

		// Addressbook section
		Addressbook: AddressbookConfig{
			DefaultURL:    "http://reg.i2p/hosts.txt",
			Subscriptions: "",
		},

		// Limits section
		Limits: LimitsConfig{
			TransitTunnels: 5000,
			OpenFiles:      0,
			CoreSize:       0,
		},

		// Trust section
		Trust: TrustConfig{
			Enabled: false,
			Family:  "",
			Routers: "",
			Hidden:  false,
		},

		// Exploratory section
		Exploratory: ExploratoryConfig{
			Inbound: TunnelConfig{
				Length:   2,
				Quantity: 3,
			},
			Outbound: TunnelConfig{
				Length:   2,
				Quantity: 3,
			},
		},

		// Persist section
		Persist: PersistConfig{
			Profiles:    true,
			Addressbook: true,
		},

		// CPU Extensions section
		CPUExt: CPUExtConfig{
			AESNI: true,
			AVX:   true,
			Force: false,
		},
	}
}

func GenerateDefaultRouterConfig(routerID int) (string, error) {
	log.WithField("routerID", routerID).Debug("Starting i2pd router config generation")

	// Initialize default configuration
	config := GenerateDefaultI2PDConfig()

	// Modify configuration as needed
	//config.Daemon = false
	//config.IPv6 = false
	//config.SSU = false
	//config.Notransit = true
	//config.NTCP2.Enabled = true
	//config.NTCP2.Published = true
	//config.NTCP2.Port = 4567 + routerID // Assign a unique port per router
	//config.HTTP.Address = "0.0.0.0"
	//config.HTTP.Port = 7070 + routerID // Unique port for web console if needed

	// Set reseed options appropriate for testnet
	//config.Reseed.Verify = false
	//config.Reseed.URLs = ""
	//config.Reseed.Threshold = 0

	// Create an INI file from the struct
	iniFile := ini.Empty()
	err := iniFile.ReflectFrom(config)
	if err != nil {
		log.WithError(err).Error("Failed to reflect config struct to INI file")
		return "", err
	}

	// Write INI file to a string
	var buffer bytes.Buffer
	_, err = iniFile.WriteTo(&buffer)
	if err != nil {
		log.WithError(err).Error("Failed to write INI file to buffer")
		return "", err
	}

	configData := buffer.String()

	log.WithFields(map[string]interface{}{
		"routerID": routerID,
		"config":   configData,
	}).Debug("i2pd router configuration generated successfully")

	return configData, nil
}

func CopyConfigToVolume(cli *client.Client, ctx context.Context, volumeName string, configData string) error {
	// Create a temporary container to copy data into the volume
	log.WithField("volumeName", volumeName).Debug("Starting config copy to volume")

	tempContainerConfig := &container.Config{
		Image:      "alpine",
		Tty:        false,
		WorkingDir: "/var/lib/i2pd",
		Cmd:        []string{"sh", "-c", "mkdir -p /var/lib/i2pd && sleep 1d"},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/var/lib/i2pd", volumeName),
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
		removeOptions := container.RemoveOptions{Force: true}
		err := cli.ContainerRemove(ctx, resp.ID, removeOptions)
		if err != nil {
			log.WithError(err).Error("Failed to remove temporary container")
		}
	}()

	// Start the container
	log.WithField("containerID", resp.ID).Debug("Starting temporary container")
	startOptions := container.StartOptions{}
	if err := cli.ContainerStart(ctx, resp.ID, startOptions); err != nil {
		log.WithError(err).Error("Failed to start temporary container")
		return fmt.Errorf("error starting temporary container: %v", err)
	}

	// Copy the configuration file into the container
	log.Debug("Creating tar archive of config data")
	tarReader, err := utils.CreateTarArchive("i2pd.conf", configData)
	if err != nil {
		log.WithError(err).Error("Failed to create tar archive")
		return fmt.Errorf("error creating tar archive: %v", err)
	}

	// Copy to the container's volume-mounted directory
	log.WithField("containerID", resp.ID).Debug("Copying config to container")
	err = cli.CopyToContainer(ctx, resp.ID, "/var/lib/i2pd", tarReader, container.CopyToContainerOptions{})
	if err != nil {
		log.WithError(err).Error("Failed to copy config to container")
		return fmt.Errorf("error copying to container: %v", err)
	}

	// Stop the container
	log.WithField("containerID", resp.ID).Debug("Stopping temporary container")
	stopOptions := container.StopOptions{}
	if err := cli.ContainerStop(ctx, resp.ID, stopOptions); err != nil {
		log.WithError(err).Error("Failed to stop temporary container")
		return fmt.Errorf("error stopping temporary container: %v", err)
	}

	log.Debug("Successfully copied config to volume")
	return nil
}
