package i2pd

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
