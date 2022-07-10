package xenstat

// #cgo pkg-config: xenstat
// #include <xenstat.h>
import "C"
import (
	"errors"
	"sync"
)

// ErrDisconnected happens when xend disconnects.
var ErrDisconnected = errors.New("not connected to xend")

// ErrCannotConnect happens when xend is not available.
var ErrCannotConnect = errors.New("cannot connect to xend")

type DomainState string

const (
	Dying    DomainState = "dying"
	Shutdown DomainState = "shutdown"
	Blocked  DomainState = "blocked"
	Crashed  DomainState = "crashed"
	Paused   DomainState = "paused"
	Running  DomainState = "running"
)

// DomainInfo represents a snapshot of numeric information about a Xen domain.
type DomainInfo struct {
	// Name is the domain name.
	Name string
	// State represents in which state the domain is.
	State DomainState
	// CPUSeconds is the total amount of CPU-seconds taken by execution of the domain since it started.
	CPUSeconds float64
	// NumVCPUs is the number of CPUs assigned to the domain.
	NumVCPUs uint32
	// MemoryBytes is the current consumption of memory of the domain.
	MemoryBytes uint64
	// MaxMemBytes is the maximum allocatable memory for the domain.
	MaxmemBytes uint64
	// NumVBDs is the number of virtual block devices assigned to the domain.
	NumVBDs uint32
	// VBD_OutOfRequests is the count of out-of-request events for the domain.
	VBD_OutOfRequests uint64
	// VBD_ReadRequests is the count of read requests for the domain.
	VBD_ReadRequests uint64
	// VBD_ReadRequests is the count of write requests for the domain.
	VBD_WriteRequests uint64
	// VBD_BytesRead is the total number of bytes read across all block devices of the domain.
	VBD_BytesRead uint64
	// VBD_BytesWritten is the total number of bytes read across all block devices of the domain.
	VBD_BytesWritten uint64
	// NumNICs is the number of Xen-virtual network interfaces assigned to the domain.
	NumNICs uint32
	// NIC_BytesTransmitted is the total number of bytes sent by the virtual NICs of the domain.
	NIC_BytesTransmitted uint64
	// NIC_BytesTransmitted is the total number of bytes received by the virtual NICs of the domain.
	NIC_BytesReceived uint64
}

type vbdT int

const (
	f_VBD_OO vbdT = iota
	f_VBD_RD
	f_VBD_WR
	f_VBD_RSECT
	f_VBD_WSECT
)

type netT int

const (
	f_NET_TX netT = iota
	f_NET_RX
)

func tot_net_bytes(domain *C.xenstat_domain, t netT) uint64 {
	num_nics := uint32(C.xenstat_domain_num_networks(domain))
	var agg C.ulonglong
	var i uint32
	var v *C.xenstat_network
	for i = 0; i < num_nics; i++ {
		v = C.xenstat_domain_network(domain, C.uint(i))
		switch t {
		case f_NET_RX:
			agg = agg + C.xenstat_network_rbytes(v)
		case f_NET_TX:
			agg = agg + C.xenstat_network_tbytes(v)
		}
	}
	return uint64(agg)
}

func tot_vbd_reqs(domain *C.xenstat_domain, t vbdT) uint64 {
	num_vbds := uint32(C.xenstat_domain_num_vbds(domain))
	var agg C.ulonglong
	var i uint32
	var v *C.xenstat_vbd
	for i = 0; i < num_vbds; i++ {
		v = C.xenstat_domain_vbd(domain, C.uint(i))
		switch t {
		case f_VBD_OO:
			agg = agg + C.xenstat_vbd_oo_reqs(v)
		case f_VBD_RD:
			agg = agg + C.xenstat_vbd_rd_reqs(v)
		case f_VBD_WR:
			agg = agg + C.xenstat_vbd_wr_reqs(v)
		case f_VBD_RSECT:
			agg = agg + C.xenstat_vbd_rd_sects(v)
		case f_VBD_WSECT:
			agg = agg + C.xenstat_vbd_wr_sects(v)
		}
	}
	return uint64(agg)
}

// XenStats represents a connection to the xend service which permits
// retrieval of statistics from the running Xen domains.
type XenStats struct {
	handle *C.xenstat_handle
	mu     sync.Mutex
}

// NewXenStats connects to the xend service.  If xend is not available,
// you'll get ErrCannotConnect.
//
// Users must call Close() after they are done with the returned XenStats
// instance.
func NewXenStats() (*XenStats, error) {
	handle := C.xenstat_init()
	if handle == nil {
		return nil, ErrCannotConnect
	}
	return &XenStats{handle: handle}, nil
}

// Close() releases resources associated with the XenStats instance.
func (x *XenStats) Close() {
	x.mu.Lock()
	defer x.mu.Unlock()
	defer C.xenstat_uninit(x.handle)
	x.handle = nil
}

// Poll returns a list of DomainInfo.
//
// If there was an error talking to xend, ErrDisconnected is returned.
// You can no longer use this instance of XenStats, and will have to
// instantiate a new one from your calling code.
//
// This code is thread-safe.
func (x *XenStats) Poll() ([]DomainInfo, error) {
	if x.handle == nil {
		return nil, ErrDisconnected
	}

	x.mu.Lock()
	defer x.mu.Unlock()

	cur_node := C.xenstat_get_node(x.handle, C.XENSTAT_ALL)
	if cur_node == nil {
		C.xenstat_uninit(x.handle)
		x.handle = nil
		return nil, ErrDisconnected
	}
	defer C.xenstat_free_node(cur_node)

	num_domains := C.xenstat_node_num_domains(cur_node)

	domains := []*C.xenstat_domain{}
	var i C.uint = 0
	for i = 0; i < num_domains; i++ {
		d := C.xenstat_node_domain_by_index(cur_node, i)
		if d == nil {
			C.xenstat_uninit(x.handle)
			x.handle = nil
			return nil, ErrDisconnected
		}
		domains = append(domains, d)
	}

	domaindata := []DomainInfo{}

	for _, domain := range domains {
		cname := C.xenstat_domain_name(domain)
		name := C.GoString(cname)
		var state DomainState
		if C.xenstat_domain_dying(domain) != 0 {
			state = Dying
		}
		if C.xenstat_domain_shutdown(domain) != 0 {
			state = Shutdown
		}
		if C.xenstat_domain_blocked(domain) != 0 {
			state = Blocked
		}
		if C.xenstat_domain_crashed(domain) != 0 {
			state = Crashed
		}
		if C.xenstat_domain_paused(domain) != 0 {
			state = Paused
		}
		if C.xenstat_domain_running(domain) != 0 {
			state = Running
		}
		domaindata = append(domaindata, DomainInfo{
			name,
			state,
			float64(uint64(C.xenstat_domain_cpu_ns(domain))) / 1000 / 1000 / 1000,
			uint32(C.xenstat_domain_num_vcpus(domain)),
			uint64(C.xenstat_domain_cur_mem(domain)),
			uint64(C.xenstat_domain_max_mem(domain)),
			uint32(C.xenstat_domain_num_vbds(domain)),
			uint64(tot_vbd_reqs(domain, f_VBD_OO)),
			uint64(tot_vbd_reqs(domain, f_VBD_RD)),
			uint64(tot_vbd_reqs(domain, f_VBD_WR)),
			uint64(tot_vbd_reqs(domain, f_VBD_RSECT) * 512),
			uint64(tot_vbd_reqs(domain, f_VBD_WSECT) * 512),
			uint32(C.xenstat_domain_num_networks(domain)),
			uint64(tot_net_bytes(domain, f_NET_TX)),
			uint64(tot_net_bytes(domain, f_NET_RX)),
		},
		)
	}

	return domaindata, nil
}
