package ipam

// Response is the response as should be written back to stage 0 clients who made an IPAM request
type Response struct {
	IPAddresses   IPAddresses `json:"ip_addresses"`
	DNSServers    []string    `json:"dns_servers,omitempty"`
	NTPServers    []string    `json:"ntp_servers,omitempty"`
	SyslogServers []string    `json:"syslog_servers,omitempty"`
	Stage1URL     string      `json:"stage1_url"`
}

// IPAddress hold all information to configure an interface on a target device.
// It maps an interface name to a list of IPaddresses with their respective netmasks (must be parseable to `net.IPNet`)
type IPAddresses map[string]IPAddress

// IPAddress hold the IP addressing information per interface including all the IP/CIDR and additional subnets that
// should be routed over the same interface (which is necessary to work with Kubernetes pods and services networks)
type IPAddress struct {
	IPAddresses []string
	VLAN        uint16
	Routes      []string
}
