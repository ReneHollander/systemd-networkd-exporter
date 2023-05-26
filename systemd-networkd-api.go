package main

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"syscall"
	"time"
	"unsafe"

	"github.com/godbus/dbus/v5"
	"github.com/golang/glog"
)

type Connection struct {
	dbusConn    *dbus.Conn
	network1Obj dbus.BusObject
}

func NewConnection(dbusConn *dbus.Conn) *Connection {
	return &Connection{
		dbusConn:    dbusConn,
		network1Obj: dbusConn.Object("org.freedesktop.network1", "/org/freedesktop/network1"),
	}
}

func parseClientId(clientIdRaw []uint8) string {
	clientIdType := clientIdRaw[0]
	clientId := clientIdRaw[1:]
	switch {
	case clientIdType == 0:
		return string(clientId)
	case clientIdType == 1:
		return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", clientId[0], clientId[1], clientId[2], clientId[3], clientId[4], clientId[5])
	case clientIdType >= 2 && clientIdType <= 254:
		return "ARP/LL"
	case clientIdType == 255:
		return "IAID/DUID"
	default:
		// this can't happen...
		panic("unknown client type")
	}
}

func convertBoottimeTimestampUsecToTime(tsUsec uint64) time.Time {
	ts := int64(tsUsec * 1e3)
	var boottime syscall.Timespec
	syscall.Syscall(syscall.SYS_CLOCK_GETTIME, 7 /* CLOCK_BOOTTIME */, uintptr(unsafe.Pointer(&boottime)), 0)
	return time.Now().Add(time.Duration(ts - boottime.Nano()))
}

type dBusNativeLink struct {
	Idx     int32
	IfName  string
	Path    dbus.ObjectPath
	linkObj dbus.BusObject
}

type dBusNativeLease struct {
	Family_Raw         uint32
	ClientId_Raw       []byte
	Address_Raw        []byte
	Gateway_Raw        []byte
	ChAddress_Raw      []byte
	ExpirationUSec_Raw uint64

	Address    *netip.Addr `dbus:"-"`
	ClientId   string      `dbus:"-"`
	Expiration time.Time   `dbus:"-"`
}

func (m *Connection) listLinks() (map[int32]dBusNativeLink, error) {
	var response []dBusNativeLink
	err := m.network1Obj.Call("org.freedesktop.network1.Manager.ListLinks", 0).Store(&response)
	if err != nil {
		return nil, err
	}
	links := make(map[int32]dBusNativeLink, len(response))
	for i := range response {
		link := &response[i]
		link.linkObj = m.dbusConn.Object("org.freedesktop.network1", link.Path)
		links[link.Idx] = *link
	}
	return links, nil
}

func (l dBusNativeLink) leases() ([]dBusNativeLease, error) {
	var leases []dBusNativeLease
	err := l.linkObj.StoreProperty("org.freedesktop.network1.DHCPServer.Leases", &leases)
	if err != nil {
		// We can't differentiate between there being an error communicating with systemd-networkd over DBus,
		// or if the Link just does not have a DHCPServer configured.
		return nil, nil
	}
	for i := range leases {
		lease := &leases[i]
		if lease.Family_Raw == syscall.AF_INET || lease.Family_Raw == syscall.AF_INET6 {
			addr, ok := netip.AddrFromSlice(lease.Address_Raw)
			if !ok {
				glog.Warningf("error parsing bytes %v with family %x into ip address", lease.Address_Raw, lease.Family_Raw)
			} else {
				lease.Address = &addr
			}
			lease.Expiration = convertBoottimeTimestampUsecToTime(lease.ExpirationUSec_Raw)
		} else {
			glog.Warningf("unknown address family %x", lease.Family_Raw)
		}
		lease.ClientId = parseClientId(lease.ClientId_Raw)
	}
	return leases, nil
}

type Lease struct {
	ClientId_Raw       []byte      `json:"ClientId"`
	ClientId           string      `json:"-"`
	Address_Raw        []byte      `json:"Address"`
	Address            *netip.Addr `json:"-"`
	Hostname           string
	ExpirationUSec_Raw uint64    `json:"ExpirationUSec"`
	Expiration         time.Time `json:"-"`
}

type StaticLease struct {
	ClientId_Raw []byte      `json:"ClientId"`
	ClientId     string      `json:"-"`
	Address_Raw  []byte      `json:"Address"`
	Address      *netip.Addr `json:"-"`
}

type DHCPServer struct {
	PoolSize     *uint32
	PoolOffset   *uint32
	Leases       []Lease
	StaticLeases []StaticLease
}

type Interface struct {
	Name       string
	Index      int32
	DHCPServer *DHCPServer
}

type Response struct {
	Interfaces []Interface
}

func (m *Connection) Describe() (Response, error) {
	var out string
	err := m.network1Obj.Call("org.freedesktop.network1.Manager.Describe", 0).Store(&out)
	if err != nil {
		return Response{}, err
	}
	var response Response
	err = json.Unmarshal([]byte(out), &response)

	nativeLinks, err := m.listLinks()
	if err != nil {
		return Response{}, err
	}

	for _, iface := range response.Interfaces {
		if iface.DHCPServer == nil {
			// If DHCPServer object cannot be found in the JSON representation, use the DBus methods to get information about leases.
			// The DHCPServer information export through the JSON is very new and might not be available on the system yet.
			nativeLeases, err := nativeLinks[iface.Index].leases()
			if err != nil {
				return Response{}, err
			}

			iface.DHCPServer = &DHCPServer{}
			for _, nativeLease := range nativeLeases {
				iface.DHCPServer.Leases = append(iface.DHCPServer.Leases, Lease{
					ClientId:   nativeLease.ClientId,
					Address:    nativeLease.Address,
					Expiration: nativeLease.Expiration,
				})
			}
		} else {
			for i := range iface.DHCPServer.Leases {
				lease := &iface.DHCPServer.Leases[i]
				lease.ClientId = parseClientId(lease.ClientId_Raw)
				addr, ok := netip.AddrFromSlice(lease.Address_Raw)
				if !ok {
					glog.Warningf("error parsing bytes %v into ip address", lease.Address_Raw)
				} else {
					lease.Address = &addr
				}
				lease.Expiration = convertBoottimeTimestampUsecToTime(lease.ExpirationUSec_Raw)
			}
			for i := range iface.DHCPServer.StaticLeases {
				lease := &iface.DHCPServer.StaticLeases[i]
				lease.ClientId = parseClientId(lease.ClientId_Raw)
				addr, ok := netip.AddrFromSlice(lease.Address_Raw)
				if !ok {
					glog.Warningf("error parsing bytes %v into ip address", lease.Address_Raw)
				} else {
					lease.Address = &addr
				}
			}
		}
	}

	return response, err
}
