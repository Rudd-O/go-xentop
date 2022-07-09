package main

// #cgo pkg-config: xenstat
// #include <xenstat.h>
import "C"
import "errors"

type DomainInfo struct {
	Name                 string
	State                string
	CPU                  float64
	NumVCPUs             uint32
	Memory               uint64
	Maxmem               uint64
	VBDCount             uint32
	VBD_OutOfRequests    uint64
	VBD_ReadRequests     uint64
	VBD_WriteRequests    uint64
	VBD_BytesRead        uint64
	VBD_BytesWritten     uint64
	NICCount             uint32
	NIC_BytesTransmitted uint64
	NIC_BytesReceived    uint64
}

type VBDT int

const (
	FIELD_VBD_OO VBDT = iota
	FIELD_VBD_RD
	FIELD_VBD_WR
	FIELD_VBD_RSECT
	FIELD_VBD_WSECT
)

var ErrDisconnected = errors.New("not connected to xend")
var ErrXenError = errors.New("error talking to xend")

type NETT int

const (
	FIELD_NET_TX NETT = iota
	FIELD_NET_RX
)

func tot_net_bytes(domain *C.xenstat_domain, t NETT) uint64 {
	num_nics := uint32(C.xenstat_domain_num_networks(domain))
	var agg C.ulonglong
	var i uint32
	var v *C.xenstat_network
	for i = 0; i < num_nics; i++ {
		v = C.xenstat_domain_network(domain, C.uint(i))
		switch t {
		case FIELD_NET_RX:
			agg = agg + C.xenstat_network_rbytes(v)
		case FIELD_NET_TX:
			agg = agg + C.xenstat_network_tbytes(v)
		}
	}
	return uint64(agg)
}

func tot_vbd_reqs(domain *C.xenstat_domain, t VBDT) uint64 {
	num_vbds := uint32(C.xenstat_domain_num_vbds(domain))
	var agg C.ulonglong
	var i uint32
	var v *C.xenstat_vbd
	for i = 0; i < num_vbds; i++ {
		v = C.xenstat_domain_vbd(domain, C.uint(i))
		switch t {
		case FIELD_VBD_OO:
			agg = agg + C.xenstat_vbd_oo_reqs(v)
		case FIELD_VBD_RD:
			agg = agg + C.xenstat_vbd_rd_reqs(v)
		case FIELD_VBD_WR:
			agg = agg + C.xenstat_vbd_wr_reqs(v)
		case FIELD_VBD_RSECT:
			agg = agg + C.xenstat_vbd_rd_sects(v)
		case FIELD_VBD_WSECT:
			agg = agg + C.xenstat_vbd_wr_sects(v)
		}
	}
	return uint64(agg)
}

type XenStats struct {
	handle *C.xenstat_handle
}

func NewXenStats() (*XenStats, error) {
	handle := C.xenstat_init()
	if handle == nil {
		return nil, ErrXenError
	}
	return &XenStats{handle}, nil
}

func (x *XenStats) Close() {
	defer C.xenstat_uninit(x.handle)
	x.handle = nil
}

func (x *XenStats) Poll() ([]DomainInfo, error) {
	if x.handle == nil {
		return nil, ErrDisconnected
	}

	cur_node := C.xenstat_get_node(x.handle, C.XENSTAT_ALL)
	if cur_node == nil {
		return nil, ErrXenError
	}
	defer C.xenstat_free_node(cur_node)

	num_domains := C.xenstat_node_num_domains(cur_node)

	domains := []*C.xenstat_domain{}
	var i C.uint = 0
	for i = 0; i < num_domains; i++ {
		d := C.xenstat_node_domain_by_index(cur_node, i)
		if d == nil {
			return nil, ErrXenError
		}
		domains = append(domains, d)
	}

	domaindata := []DomainInfo{}

	for _, domain := range domains {
		cname := C.xenstat_domain_name(domain)
		name := C.GoString(cname)
		var state string
		if C.xenstat_domain_dying(domain) != 0 {
			state = "dying"
		}
		if C.xenstat_domain_shutdown(domain) != 0 {
			state = "shutdown"
		}
		if C.xenstat_domain_blocked(domain) != 0 {
			state = "blocked"
		}
		if C.xenstat_domain_crashed(domain) != 0 {
			state = "crashed"
		}
		if C.xenstat_domain_paused(domain) != 0 {
			state = "paused"
		}
		if C.xenstat_domain_running(domain) != 0 {
			state = "running"
		}
		domaindata = append(domaindata, DomainInfo{
			name,
			state,
			float64(uint64(C.xenstat_domain_cpu_ns(domain))) / 1000 / 1000 / 1000,
			uint32(C.xenstat_domain_num_vcpus(domain)),
			uint64(C.xenstat_domain_cur_mem(domain)),
			uint64(C.xenstat_domain_max_mem(domain)),
			uint32(C.xenstat_domain_num_vbds(domain)),
			uint64(tot_vbd_reqs(domain, FIELD_VBD_OO)),
			uint64(tot_vbd_reqs(domain, FIELD_VBD_RD)),
			uint64(tot_vbd_reqs(domain, FIELD_VBD_WR)),
			uint64(tot_vbd_reqs(domain, FIELD_VBD_RSECT) * 512),
			uint64(tot_vbd_reqs(domain, FIELD_VBD_WSECT) * 512),
			uint32(C.xenstat_domain_num_networks(domain)),
			uint64(tot_net_bytes(domain, FIELD_NET_TX)),
			uint64(tot_net_bytes(domain, FIELD_NET_RX)),
		},
		)
	}

	return domaindata, nil
}
