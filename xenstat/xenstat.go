package xenstat

// #cgo pkg-config: xenstat
// #include <xenstat.h>
import "C"
import (
	"errors"
	"fmt"
	"log"
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

type VBDInfo struct {
	// Major is the major block device number.
	Major uint8
	// Minor is the minor block device number.
	Minor uint8
	// OutOfRequests is the count of out-of-request events for this device.
	OutOfRequests uint64
	// ReadRequests is the count of read requests for this device.
	ReadRequests uint64
	// ReadRequests is the count of write requests for this device.
	WriteRequests uint64
	// BytesRead is the total number of bytes read by this device.
	BytesRead uint64
	// BytesWritten is the total number of bytes read by this device.
	BytesWritten uint64
}

type NICInfo struct {
	// BytesTransmitted is the total number of bytes sent by this virtual NIC.
	BytesTransmitted uint64
	// BytesTransmitted is the total number of bytes received by this virtual NIC.
	BytesReceived uint64
}

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
	// NumNICs is the number of Xen-virtual network interfaces assigned to the domain.
	NumNICs uint32
	// VBDs contains a list of VBDInfo that disaggregates the statistics for each virtual block device.
	VBDs []VBDInfo
	// NICs contains a list of NICInfo that disaggregates the statistics for each virtual network device.
	NICs []NICInfo
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

func dev_net_bytes(domain *C.xenstat_domain, t netT, devid uint32) uint64 {
	var v *C.xenstat_network
	v = C.xenstat_domain_network(domain, C.uint(devid))
	switch t {
	case f_NET_RX:
		return uint64(C.xenstat_network_rbytes(v))
	case f_NET_TX:
		return uint64(C.xenstat_network_tbytes(v))
	}
	panic("wrong case")
}

func dev_vbd_reqs(domain *C.xenstat_domain, t vbdT, devid uint32) (uint64, error) {
	var v *C.xenstat_vbd
	v = C.xenstat_domain_vbd(domain, C.uint(devid))
	if v == nil {
		return 0, fmt.Errorf("could not get VBD %d type %v from domain %+v", devid, t, domain)
	}
	switch t {
	case f_VBD_OO:
		return uint64(C.xenstat_vbd_oo_reqs(v)), nil
	case f_VBD_RD:
		return uint64(C.xenstat_vbd_rd_reqs(v)), nil
	case f_VBD_WR:
		return uint64(C.xenstat_vbd_wr_reqs(v)), nil
	case f_VBD_RSECT:
		return uint64(C.xenstat_vbd_rd_sects(v)), nil
	case f_VBD_WSECT:
		return uint64(C.xenstat_vbd_wr_sects(v)), nil
	}
	panic("wrong case")
}

func dev_vbd_major_minor(domain *C.xenstat_domain, devid uint32) (uint8, uint8, error) {
	var v *C.xenstat_vbd
	v = C.xenstat_domain_vbd(domain, C.uint(devid))
	if v == nil {
		return 0, 0, fmt.Errorf("could not get VBD %d major/minor from domain %+v", devid, domain)
	}
	var dev C.uint = C.xenstat_vbd_dev(v)
	var major uint8 = uint8(255 & (dev >> 8))
	var minor uint8 = uint8(255 & dev)
	return major, minor, nil
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

		num_vbds := uint32(C.xenstat_domain_num_vbds(domain))
		num_nics := uint32(C.xenstat_domain_num_networks(domain))

		var i uint32
		var vv []VBDInfo
		var nn []NICInfo
		for i = 0; i < num_vbds; i++ {
			outOfRequests, err := dev_vbd_reqs(domain, f_VBD_OO, i)
			if err != nil {
				log.Printf("%s: %s", name, err)
				continue
			}
			readRequests, err := dev_vbd_reqs(domain, f_VBD_RD, i)
			if err != nil {
				log.Printf("%s: %s", name, err)
				continue
			}
			writeRequests, err := dev_vbd_reqs(domain, f_VBD_WR, i)
			if err != nil {
				log.Printf("%s: %s", name, err)
				continue
			}
			sectorsRead, err := dev_vbd_reqs(domain, f_VBD_RSECT, i)
			if err != nil {
				log.Printf("%s: %s", name, err)
				continue
			}
			sectorsWritten, err := dev_vbd_reqs(domain, f_VBD_WSECT, i)
			if err != nil {
				log.Printf("%s: %s", name, err)
				continue
			}
			major, minor, err := dev_vbd_major_minor(domain, i)
			if err != nil {
				log.Printf("%s: %s", name, err)
				continue
			}
			vbdinfo := VBDInfo{
				Major:         major,
				Minor:         minor,
				OutOfRequests: outOfRequests,
				ReadRequests:  readRequests,
				WriteRequests: writeRequests,
				BytesRead:     sectorsRead * 512,
				BytesWritten:  sectorsWritten * 512,
			}
			vv = append(vv, vbdinfo)
		}
		for i = 0; i < num_nics; i++ {
			nn = append(nn, NICInfo{
				BytesTransmitted: dev_net_bytes(domain, f_NET_TX, i),
				BytesReceived:    dev_net_bytes(domain, f_NET_RX, i),
			})
		}

		domaindata = append(domaindata, DomainInfo{
			name,
			state,
			float64(uint64(C.xenstat_domain_cpu_ns(domain))) / 1000 / 1000 / 1000,
			uint32(C.xenstat_domain_num_vcpus(domain)),
			uint64(C.xenstat_domain_cur_mem(domain)),
			uint64(C.xenstat_domain_max_mem(domain)),
			uint32(C.xenstat_domain_num_vbds(domain)),
			uint32(C.xenstat_domain_num_networks(domain)),
			vv,
			nn,
		},
		)
	}

	return domaindata, nil
}
