// Package api defines shared JSON-safe RPC/IPC contracts, DTOs, and event payloads
// used across core, desktop, mobile, and the Angular UI.
package api

// PlatformTarget identifies the host runtime environment.
type PlatformTarget string

const (
	PlatformDesktop PlatformTarget = "desktop"
	PlatformAndroid PlatformTarget = "android"
)

// ConnectionStatus represents the active tunnel/bypass lifecycle state.
type ConnectionStatus string

const (
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusConnecting   ConnectionStatus = "connecting"
	StatusConnected    ConnectionStatus = "connected"
)

// HomeMethod selects the primary connection mode in Simple Mode.
type HomeMethod string

const (
	MethodAll     HomeMethod = "all"
	MethodAWG     HomeMethod = "awg"
	MethodProxy   HomeMethod = "proxy"
	MethodMASQUE  HomeMethod = "masque"
	MethodPsiphon HomeMethod = "psiphon"
	MethodNoVPN   HomeMethod = "novpn"
)

// NovpnMode selects the active VPN-less bypass mode.
type NovpnMode string

const (
	NovpnByeDPI         NovpnMode = "byedpi"
	NovpnZapret         NovpnMode = "zapret"
	NovpnSmartDNS       NovpnMode = "smartdns"
	NovpnByeDPISmartDNS NovpnMode = "byedpi_smartdns"
)

// ServiceMode selects how traffic is captured on the host.
type ServiceMode string

const (
	ServiceModeVPN         ServiceMode = "vpn"
	ServiceModeSystemProxy ServiceMode = "system-proxy"
	ServiceModeLocalProxy  ServiceMode = "local-proxy"
)

// SmartDnsMode selects how Smart DNS / Smart Hosts operates.
type SmartDnsMode string

const (
	SmartDnsProxy SmartDnsMode = "proxy"
	SmartDnsTun   SmartDnsMode = "tun"
	SmartDnsHosts SmartDnsMode = "hosts"
)

// ChainCascade defines an optional upstream cascade hop.
type ChainCascade string

const (
	ChainNone    ChainCascade = "none"
	ChainPsiphon ChainCascade = "psiphon"
	ChainWARP    ChainCascade = "warp"
)

// NodeProtoGroup groups protocol engines for filtering and UI presentation.
type NodeProtoGroup string

const (
	ProtoGroupAWG     NodeProtoGroup = "awg"
	ProtoGroupProxy   NodeProtoGroup = "proxy"
	ProtoGroupMASQUE  NodeProtoGroup = "masque"
	ProtoGroupWARP    NodeProtoGroup = "warp"
	ProtoGroupPsiphon NodeProtoGroup = "psiphon"
)

// AntiBotState indicates pre-connect anti-bot / captcha check results.
type AntiBotState string

const (
	AntiBotClean     AntiBotState = "Clean"
	AntiBotChallenge AntiBotState = "Challenge"
	AntiBotError     AntiBotState = "Error"
)

// SourceItem describes an imported subscription, local file, or built-in generator.
type SourceItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Meta     string `json:"meta"`
	URL      string `json:"url,omitempty"`
	Quota    string `json:"quota,omitempty"`
	QuotaKey string `json:"quotaKey,omitempty"`
}

// NodeItem represents a single selectable outbound node or tunnel profile.
type NodeItem struct {
	ID         string            `json:"id"`
	SourceID   string            `json:"sourceId"`
	ISO        string            `json:"iso"`
	Name       string            `json:"name"`
	Endpoint   string            `json:"endpoint"`
	Proto      string            `json:"proto"`
	ProtoGroup NodeProtoGroup    `json:"protoGroup"`
	TcpMs      *int              `json:"tcpMs"`
	UrlMs      *int              `json:"urlMs"`
	AntiBot    AntiBotState      `json:"antiBot"`
	Probing    bool              `json:"probing"`
	MTU        int               `json:"mtu"`
	Params     map[string]string `json:"params"`
	RawConfig  string            `json:"rawConfig,omitempty"`
}

// WarpEndpoint describes a scanned Cloudflare WARP endpoint.
type WarpEndpoint struct {
	Addr   string `json:"addr"`
	Region string `json:"region"`
	RTT    int    `json:"rtt"`
	Loss   string `json:"loss"`
}

// ImportDetectionResult summarizes detected protocol metadata for raw import text.
type ImportDetectionResult struct {
	Proto      string         `json:"proto"`
	ProtoGroup NodeProtoGroup `json:"protoGroup"`
	Label      string         `json:"label"`
}

// ConnectPayload is sent by UI/clients to initiate a connection.
type ConnectPayload struct {
	Mode       string `json:"mode"`
	TargetName string `json:"targetName"`
	NodeID     string `json:"nodeId,omitempty"`
	ProfileID  string `json:"profileId,omitempty"`
	Standalone bool   `json:"standalone"`
}

// ProbeOutcome holds pre-connect latency and anti-bot results for a node.
type ProbeOutcome struct {
	NodeID  string       `json:"nodeId"`
	TcpMs   int          `json:"tcpMs"`
	UrlMs   *int         `json:"urlMs"`
	AntiBot AntiBotState `json:"antiBot"`
	Error   string       `json:"error,omitempty"`
}

// RuleTier classifies a routing rule's matching layer.
type RuleTier string

const (
	RuleTierApp      RuleTier = "app"
	RuleTierNetwork  RuleTier = "network"
	RuleTierCombined RuleTier = "combined"
)

// AppMatchMode controls how application matchers are evaluated.
type AppMatchMode string

const (
	AppMatchOff     AppMatchMode = "off"
	AppMatchInclude AppMatchMode = "include"
	AppMatchExclude AppMatchMode = "exclude"
)

// TransportNetwork filters L4 transport protocols.
type TransportNetwork string

const (
	TransportBoth TransportNetwork = "both"
	TransportTCP  TransportNetwork = "tcp"
	TransportUDP  TransportNetwork = "udp"
)

// IpVersionFilter restricts rule matching by IP family.
type IpVersionFilter string

const (
	IpVerAny  IpVersionFilter = "any"
	IpVerIPv4 IpVersionFilter = "ipv4"
	IpVerIPv6 IpVersionFilter = "ipv6"
)

// InboundFilter restricts rule matching by inbound listener.
type InboundFilter string

const (
	InboundAny   InboundFilter = "any"
	InboundTun   InboundFilter = "tun"
	InboundProxy InboundFilter = "proxy"
	InboundLAN   InboundFilter = "lan"
)

// RoutingRule defines one ordered rule in the visual routing constructor.
type RoutingRule struct {
	ID                string           `json:"id"`
	PresetID          string           `json:"presetId,omitempty"`
	Name              string           `json:"name"`
	Tier              RuleTier         `json:"tier"`
	Enabled           bool             `json:"enabled"`
	Invert            bool             `json:"invert"`
	ContinueToNetwork bool             `json:"continueToNetwork"`
	AppMode           AppMatchMode     `json:"appMode"`
	Apps              []string         `json:"apps"`
	AndroidApps       []string         `json:"androidApps"`
	AppPaths          []string         `json:"appPaths"`
	Domains           []string         `json:"domains"`
	Suffixes          []string         `json:"suffixes"`
	Keywords          []string         `json:"keywords"`
	Regex             string           `json:"regex"`
	Lists             []string         `json:"lists"`
	CIDRs             []string         `json:"cidrs"`
	IPPools           []string         `json:"ipPools"`
	GeoIP             []string         `json:"geoip"`
	Network           TransportNetwork `json:"network"`
	Ports             []string         `json:"ports"`
	L7                []string         `json:"l7"`
	IpVer             IpVersionFilter  `json:"ipVer"`
	Inbound           InboundFilter    `json:"inbound"`
	Target            string           `json:"target"`
}

// RoutingProfile holds a user-defined ordered list of routing rules.
// A newly created profile MUST have an empty Rules slice (zero hidden defaults).
type RoutingProfile struct {
	ID            string        `json:"id"`
	NameRu        string        `json:"nameRu"`
	NameEn        string        `json:"nameEn"`
	DefaultTarget string        `json:"defaultTarget"`
	Rules         []RoutingRule `json:"rules"`
}

// StickyHop represents a node in a cross-protocol sticky-fallback chain.
type StickyHop struct {
	NodeID string `json:"nodeId"`
	Label  string `json:"label"`
	Proto  string `json:"proto"`
	Failed bool   `json:"failed"`
}

// ZapretStrategy defines DPI bypass rules for YouTube, QUIC UDP 443, and Discord.
type ZapretStrategy struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	YT      string `json:"yt"`
	QUIC    string `json:"quic"`
	Discord string `json:"discord"`
}

// ConflictAction describes a deterministic user resolution for a network conflict.
type ConflictAction struct {
	Label   string `json:"label"`
	Action  string `json:"action"`
	Primary bool   `json:"primary"`
}

// ConflictReport surfaces conflicting active network layers (TUN vs hosts, WinDivert, etc.).
type ConflictReport struct {
	ID      string           `json:"id"`
	Title   string           `json:"title"`
	Detail  string           `json:"detail"`
	Actions []ConflictAction `json:"actions"`
}

// LogLevel identifies log entry severity.
type LogLevel string

const (
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

// LogCategory identifies the subsystem emitting a log entry.
type LogCategory string

const (
	LogCatCore   LogCategory = "core"
	LogCatRouter LogCategory = "router"
	LogCatProbe  LogCategory = "probe"
	LogCatDPI    LogCategory = "dpi"
)

// LogEntry represents a structured log line streamed to the UI.
type LogEntry struct {
	Time string      `json:"time"`
	Lvl  LogLevel    `json:"lvl"`
	Cat  LogCategory `json:"cat"`
	Msg  string      `json:"msg"`
}

// AppSettings stores persistent user preferences.
type AppSettings struct {
	Theme           string `json:"theme"`
	Lang            string `json:"lang"`
	ServiceMode     string `json:"serviceMode"`
	SmartDnsMode    string `json:"smartDnsMode"`
	Ipv6Mode        string `json:"ipv6Mode"`
	Balancer        string `json:"balancer"`
	ActiveNodeID    string `json:"activeNodeId,omitempty"`
	ActiveProfileID string `json:"activeProfileId,omitempty"`
}
