package main

import (
	"strconv"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	conn *Connection

	dhcpServerConfigured  *prometheus.Desc
	dhcpv4Leases          *prometheus.Desc
	dhcpv4LeaseExpiration *prometheus.Desc
	dhcpv4StaticLeases    *prometheus.Desc
	dhcpv6Leases          *prometheus.Desc
	dhcpv6LeaseExpiration *prometheus.Desc
	dhcpv6StaticLeases    *prometheus.Desc
}

func NewCollector(conn *Connection) *Collector {
	return &Collector{conn: conn,
		dhcpServerConfigured: prometheus.NewDesc(
			"systemd_networkd_dhcpserver_configured",
			"Set to 1 if a DHCP Server is configured for the interface.",
			[]string{"interface", "pool_size", "pool_offset"}, nil,
		),
		dhcpv4Leases: prometheus.NewDesc(
			"systemd_networkd_dhcpserver_ipv4_lease",
			"Current IPv4 DHCP Leases offered by the DHCP Server.",
			[]string{"interface",
				"ip_address",
				"client_id", "hostname"}, nil,
		), dhcpv4LeaseExpiration: prometheus.NewDesc(
			"systemd_networkd_dhcpserver_ipv4_lease_expiration_time",
			"UNIX timestamp (in seconds) at which point the lease expires.",
			[]string{"interface",
				"ip_address",
				"client_id", "hostname"}, nil,
		), dhcpv4StaticLeases: prometheus.NewDesc(
			"systemd_networkd_dhcpserver_ipv4_static_lease",
			"Static IPv4 DHCP Leases offered by the DHCP Server.",
			[]string{"interface",
				"ip_address",
				"client_id"}, nil,
		), dhcpv6Leases: prometheus.NewDesc(
			"systemd_networkd_dhcpserver_ipv6_lease",
			"Current IPv6 DHCP Leases offered by the DHCP Server.",
			[]string{"interface",
				"ip_address",
				"client_id", "hostname"}, nil,
		), dhcpv6LeaseExpiration: prometheus.NewDesc(
			"systemd_networkd_dhcpserver_ipv6_lease_expiration_time",
			"UNIX timestamp (in seconds) at which point the lease expires.",
			[]string{"interface",
				"ip_address",
				"client_id", "hostname"}, nil,
		), dhcpv6StaticLeases: prometheus.NewDesc(
			"systemd_networkd_dhcpserver_ipv6_static_lease",
			"Static IPv6 DHCP Leases offered by the DHCP Server.",
			[]string{"interface",
				"ip_address",
				"client_id"}, nil,
		)}
}

// Describe returns all descriptions of the collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.dhcpServerConfigured
	ch <- c.dhcpv4Leases
	ch <- c.dhcpv4LeaseExpiration
	ch <- c.dhcpv4StaticLeases
	ch <- c.dhcpv6Leases
	ch <- c.dhcpv6LeaseExpiration
	ch <- c.dhcpv6StaticLeases
}

// Collect returns the current state of all metrics of the collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	glog.V(1).Infof("collecting systemd-networkd metrics")

	response, err := c.conn.Describe()
	if err != nil {
		glog.Errorf("error getting systemd-networkd data via DBus: %v", err)
		return
	}

	for _, link := range response.Interfaces {
		glog.V(1).Info("processing link: ", link.Name)
		if link.DHCPServer != nil {
			poolSize := ""
			if link.DHCPServer.PoolSize != nil {
				poolSize = strconv.FormatUint(uint64(*link.DHCPServer.PoolSize), 10)
			}
			poolOffset := ""
			if link.DHCPServer.PoolOffset != nil {
				poolOffset = strconv.FormatUint(uint64(*link.DHCPServer.PoolOffset), 10)
			}

			ch <- prometheus.MustNewConstMetric(c.dhcpServerConfigured, prometheus.GaugeValue, 1, link.Name, poolSize, poolOffset)
			for _, lease := range link.DHCPServer.Leases {
				if lease.Address == nil {
					glog.Errorf("skipped exporting lease %v since it had no address", lease)
					continue
				}
				if lease.Address.Is4() {
					ch <- prometheus.MustNewConstMetric(c.dhcpv4Leases, prometheus.GaugeValue, 1, link.Name, lease.Address.String(), lease.ClientId, lease.Hostname)
					ch <- prometheus.MustNewConstMetric(c.dhcpv4LeaseExpiration, prometheus.GaugeValue, float64(lease.Expiration.Unix()), link.Name, lease.Address.String(), lease.ClientId, lease.Hostname)
				} else if lease.Address.Is6() {
					ch <- prometheus.MustNewConstMetric(c.dhcpv6Leases, prometheus.GaugeValue, 1, link.Name, lease.Address.String(), lease.ClientId, lease.Hostname)
					ch <- prometheus.MustNewConstMetric(c.dhcpv6LeaseExpiration, prometheus.GaugeValue, float64(lease.Expiration.Unix()), link.Name, lease.Address.String(), lease.ClientId, lease.Hostname)
				}
			}
			for _, lease := range link.DHCPServer.StaticLeases {
				if lease.Address == nil {
					glog.Errorf("skipped exporting lease %v since it had no address", lease)
					continue
				}
				if lease.Address.Is4() {
					ch <- prometheus.MustNewConstMetric(c.dhcpv4StaticLeases, prometheus.GaugeValue, 1, link.Name, lease.Address.String(), lease.ClientId)
				} else if lease.Address.Is6() {
					ch <- prometheus.MustNewConstMetric(c.dhcpv6StaticLeases, prometheus.GaugeValue, 1, link.Name, lease.Address.String(), lease.ClientId)
				}
			}
		}
	}
}
