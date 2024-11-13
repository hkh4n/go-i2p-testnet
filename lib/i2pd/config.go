package i2pd

import (
	"bytes"
	"context"
	"fmt"
	"gopkg.in/ini.v1"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go-i2p-testnet/lib/utils"
)

// I2PDConfig represents the complete i2pd configuration
type I2PDConfig struct {
	// Global options (before any section)
	TunnelsConf   string `ini:"tunconf"`
	TunnelsDir    string `ini:"tunnelsdir"`
	CertsDir      string `ini:"certsdir"`
	Pidfile       string `ini:"pidfile"`
	Log           string `ini:"log"`
	Logfile       string `ini:"logfile"`
	Loglevel      string `ini:"loglevel"`
	Logclftime    bool   `ini:"logclftime"`
	Daemon        bool   `ini:"daemon"`
	Family        string `ini:"family"`
	Ifname        string `ini:"ifname"`
	Ifname4       string `ini:"ifname4"`
	Ifname6       string `ini:"ifname6"`
	Address4      string `ini:"address4"`
	Address6      string `ini:"address6"`
	Host          string `ini:"host"`
	Port          int    `ini:"port"`
	IPv4          bool   `ini:"ipv4"`
	IPv6          bool   `ini:"ipv6"`
	SSU           bool   `ini:"ssu"`
	Bandwidth     string `ini:"bandwidth"`
	Share         int    `ini:"share"`
	Notransit     bool   `ini:"notransit"`
	Floodfill     bool   `ini:"floodfill"`
	Service       bool   `ini:"service"`
	Datadir       string `ini:"datadir"`
	Netid         int    `ini:"netid"`
	Nat           bool   `ini:"nat"`
	ReservedRange bool   `ini:"reservedrange"`
	// Sections
	NTCP2        NTCP2Config        `ini:"ntcp2"`
	SSU2         SSU2Config         `ini:"ssu2"`
	HTTP         HTTPConfig         `ini:"http"`
	HTTPProxy    HTTPProxyConfig    `ini:"httpproxy"`
	SocksProxy   SocksProxyConfig   `ini:"socksproxy"`
	SAM          SAMConfig          `ini:"sam"`
	BOB          BOBConfig          `ini:"bob"`
	I2CP         I2CPConfig         `ini:"i2cp"`
	I2PControl   I2PControlConfig   `ini:"i2pcontrol"`
	Cryptography CryptographyConfig `ini:"cryptography"`
	UPnP         UPnPConfig         `ini:"upnp"`
	Meshnets     MeshnetsConfig     `ini:"meshnets"`
	Reseed       ReseedConfig       `ini:"reseed"`
	Addressbook  AddressbookConfig  `ini:"addressbook"`
	Limits       LimitsConfig       `ini:"limits"`
	Trust        TrustConfig        `ini:"trust"`
	Persist      PersistConfig      `ini:"persist"`
	CPUExt       CPUExtConfig       `ini:"cpuext"`
	Nettime      NettimeConfig      `ini:"nettime"`
	//LocalAddressbook LocalAddressbookConfig `ini:"localaddressbook"`
	//Windows WindowsConfig `ini:"windows"`
	//Unix    UnixConfig    `ini:"unix"`
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
	Enabled                bool   `ini:"enabled"`
	Address                string `ini:"address"`
	Port                   int    `ini:"port"`
	Keys                   string `ini:"keys"`
	AddressHelper          bool   `ini:"addresshelper"`
	Outproxy               string `ini:"outproxy"`
	SignatureType          int    `ini:"signaturetype"`
	InboundQuantity        int    `ini:"inbound.quantity"`
	OutboundLength         int    `ini:"outbound.length"`
	OutboundQuantity       int    `ini:"outbound.quantity"`
	OutboundLengthVariance int    `ini:"outbound.lengthVariance"`
	I2CPLeaseSetType       int    `ini:"i2cp.leaseSetType"`
	I2CPLeaseSetEncType    string `ini:"i2cp.leaseSetEncType"`
}

// SocksProxy Section
type SocksProxyConfig struct {
	Enabled                bool   `ini:"enabled"`
	Address                string `ini:"address"`
	Port                   int    `ini:"port"`
	Keys                   string `ini:"keys"`
	OutproxyEnabled        bool   `ini:"outproxy.enabled"`
	Outproxy               string `ini:"outproxy"`
	OutproxyPort           int    `ini:"outproxyport"`
	SignatureType          int    `ini:"signaturetype"`
	InboundQuantity        int    `ini:"inbound.quantity"`
	OutboundLength         int    `ini:"outbound.length"`
	OutboundQuantity       int    `ini:"outbound.quantity"`
	OutboundLengthVariance int    `ini:"outbound.lengthVariance"`
	I2CPLeaseSetType       int    `ini:"i2cp.leaseSetType"`
	I2CPLeaseSetEncType    string `ini:"i2cp.leaseSetEncType"`
}

// SAM Section
type SAMConfig struct {
	Enabled      bool   `ini:"enabled"`
	Address      string `ini:"address"`
	Port         int    `ini:"port"`
	SingleThread bool   `ini:"singlethread"`
}

// BOB Section
type BOBConfig struct {
	Enabled bool   `ini:"enabled"`
	Address string `ini:"address"`
	Port    int    `ini:"port"`
}

// I2CP Section
type I2CPConfig struct {
	Enabled      bool   `ini:"enabled"`
	Address      string `ini:"address"`
	Port         int    `ini:"port"`
	SingleThread bool   `ini:"singlethread"`
}

// I2PControl Section
type I2PControlConfig struct {
	Enabled  bool   `ini:"enabled"`
	Address  string `ini:"address"`
	Port     int    `ini:"port"`
	Password string `ini:"password"`
	Cert     string `ini:"cert"`
	Key      string `ini:"key"`
}

// Cryptography Section
type CryptographyConfig struct {
	Precomputation PrecomputationConfig `ini:"precomputation"`
	// Add other cryptographic options here
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
	HostsFile     string `ini:"hostsfile"`
}

// Limits Section
type LimitsConfig struct {
	TransitTunnels int     `ini:"transittunnels"`
	OpenFiles      int     `ini:"openfiles"`
	CoreSize       int     `ini:"coresize"`
	Zombies        float64 `ini:"zombies"`
}

// Trust Section
type TrustConfig struct {
	Enabled bool   `ini:"enabled"`
	Family  string `ini:"family"`
	Routers string `ini:"routers"`
	Hidden  bool   `ini:"hidden"`
}

// TunnelConfig represents inbound or outbound tunnel configurations
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

// Nettime Section
type NettimeConfig struct {
	Enabled         bool   `ini:"enabled"`
	NtpServers      string `ini:"ntpservers"`
	NtpSyncInterval int    `ini:"ntpsyncinterval"`
}

func GenerateDefaultI2PDConfig() *I2PDConfig {
	return &I2PDConfig{
		// Global options (before any section)
		TunnelsConf: "/var/lib/i2pd/tunnels.conf",
		TunnelsDir:  "/var/lib/i2pd/tunnels.d/",
		CertsDir:    "/var/lib/i2pd/certificates",
		Pidfile:     "/run/i2pd.pid",
		Log:         "stdout",
		Logfile:     "",
		Loglevel:    "debug",
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
		Floodfill:   true,
		Service:     false,
		Datadir:     "/var/lib/i2pd",
		Netid:       2,
		Nat:         true,

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
			Enabled:                true,
			Address:                "127.0.0.1",
			Port:                   4444,
			Keys:                   "",
			AddressHelper:          true,
			Outproxy:               "",
			SignatureType:          7,
			InboundQuantity:        5,
			OutboundLength:         3,
			OutboundQuantity:       5,
			OutboundLengthVariance: 0,
			I2CPLeaseSetType:       3,
			I2CPLeaseSetEncType:    "",
		},

		// SOCKS Proxy section
		SocksProxy: SocksProxyConfig{
			Enabled:                true,
			Address:                "127.0.0.1",
			Port:                   4447,
			Keys:                   "",
			OutproxyEnabled:        false,
			Outproxy:               "",
			OutproxyPort:           0,
			SignatureType:          7,
			InboundQuantity:        5,
			OutboundLength:         3,
			OutboundQuantity:       5,
			OutboundLengthVariance: 0,
			I2CPLeaseSetType:       3,
			I2CPLeaseSetEncType:    "",
		},

		// SAM section
		SAM: SAMConfig{
			Enabled:      true,
			Address:      "127.0.0.1",
			Port:         7656,
			SingleThread: true,
		},

		// BOB section
		BOB: BOBConfig{
			Enabled: false,
			Address: "127.0.0.1",
			Port:    2827,
		},

		// I2CP section
		I2CP: I2CPConfig{
			Enabled:      false,
			Address:      "127.0.0.1",
			Port:         7654,
			SingleThread: true,
		},

		// I2PControl section
		I2PControl: I2PControlConfig{
			Enabled:  false,
			Address:  "127.0.0.1",
			Port:     7650,
			Password: "itoopie",
			Cert:     "i2pcontrol.crt.pem",
			Key:      "i2pcontrol.key.pem",
		},

		// Cryptography section
		Cryptography: CryptographyConfig{
			Precomputation: PrecomputationConfig{
				ElGamal: true,
			},
			// Initialize other cryptographic options as needed
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
			Verify:    true,
			URLs:      "https://reseed.i2p-projekt.de/,https://i2p.mooo.com/netDb/,https://netdb.i2p2.no/",
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
			HostsFile:     "hosts.txt",
		},

		// Limits section
		Limits: LimitsConfig{
			TransitTunnels: 10000, // As per default
			OpenFiles:      0,
			CoreSize:       0,
			Zombies:        0.00,
		},

		// Trust section
		Trust: TrustConfig{
			Enabled: false,
			Family:  "",
			Routers: "",
			Hidden:  false,
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

		// Nettime section
		Nettime: NettimeConfig{
			Enabled:         false,
			NtpServers:      "pool.ntp.org",
			NtpSyncInterval: 72,
		},
	}
}

func GenerateRouterConfig(routerID int) (string, error) {
	log.WithField("routerID", routerID).Debug("Starting i2pd router config generation")

	// Initialize default configuration
	config := GenerateDefaultI2PDConfig()
	config.Netid = 5
	config.ReservedRange = false
	config.Nat = false
	config.Floodfill = true

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

	// Optionally, handle additional configuration files like local addressbook
	/*
	   localAddrBookData := "your local addressbook content here"
	   tarReader, err = utils.CreateTarArchive("addressbook/local.csv", localAddrBookData)
	   if err != nil {
	       log.WithError(err).Error("Failed to create tar archive for local addressbook")
	       return fmt.Errorf("error creating tar archive for local addressbook: %v", err)
	   }

	   log.WithField("containerID", resp.ID).Debug("Copying local addressbook to container")
	   err = cli.CopyToContainer(ctx, resp.ID, "/var/lib/i2pd/addressbook", tarReader, container.CopyToContainerOptions{})
	   if err != nil {
	       log.WithError(err).Error("Failed to copy local addressbook to container")
	       return fmt.Errorf("error copying local addressbook to container: %v", err)
	   }
	*/

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
