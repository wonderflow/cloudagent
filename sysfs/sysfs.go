package sysfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	blockDir   = "/sys/block"
	cacheDir   = "/sys/devices/system/cpu/cpu"
	netDir     = "/sys/class/net"
	dmiDir     = "/sys/class/dmi"
	ppcDevTree = "/proc/device-tree"
)

type CacheInfo struct {
	// size in bytes
	Size uint64
	// cache type - instruction, data, unified
	Type string
	// distance from cpus in a multi-level hierarchy
	Level int
	// number of cpus that can access this cache.
	Cpus int
}

// Abstracts the lowest level calls to sysfs.
type SysFs interface {
	// Get directory information for available block devices.
	GetBlockDevices() ([]os.FileInfo, error)
	// Get Size of a given block device.
	GetBlockDeviceSize(string) (string, error)
	// Get scheduler type for the block device.
	GetBlockDeviceScheduler(string) (string, error)
	// Get device major:minor number string.
	GetBlockDeviceNumbers(string) (string, error)

	GetNetworkDevices() ([]os.FileInfo, error)
	GetNetworkAddress(string) (string, error)
	GetNetworkMtu(string) (string, error)
	GetNetworkSpeed(string) (string, error)
	GetNetworkStatValue(dev string, stat string) (uint64, error)

	// Get directory information for available caches accessible to given cpu.
	GetCaches(id int) ([]os.FileInfo, error)
	// Get information for a cache accessible from the given cpu.
	GetCacheInfo(cpu int, cache string) (CacheInfo, error)

	GetSystemUUID() (string, error)
}

type realSysFs struct{}

func NewRealSysFs() (SysFs, error) {
	return &realSysFs{}, nil
}

func (self *realSysFs) GetBlockDevices() ([]os.FileInfo, error) {
	return ioutil.ReadDir(blockDir)
}

func (self *realSysFs) GetBlockDeviceNumbers(name string) (string, error) {
	dev, err := ioutil.ReadFile(path.Join(blockDir, name, "/dev"))
	if err != nil {
		return "", err
	}
	return string(dev), nil
}

func (self *realSysFs) GetBlockDeviceScheduler(name string) (string, error) {
	sched, err := ioutil.ReadFile(path.Join(blockDir, name, "/queue/scheduler"))
	if err != nil {
		return "", err
	}
	return string(sched), nil
}

func (self *realSysFs) GetBlockDeviceSize(name string) (string, error) {
	size, err := ioutil.ReadFile(path.Join(blockDir, name, "/size"))
	if err != nil {
		return "", err
	}
	return string(size), nil
}

func (self *realSysFs) GetNetworkDevices() ([]os.FileInfo, error) {
	files, err := ioutil.ReadDir(netDir)
	if err != nil {
		return nil, err
	}

	// Filter out non-directory & non-symlink files
	var dirs []os.FileInfo
	for _, f := range files {
		if f.Mode()|os.ModeSymlink != 0 {
			f, err = os.Stat(path.Join(netDir, f.Name()))
			if err != nil {
				continue
			}
		}
		if f.IsDir() {
			dirs = append(dirs, f)
		}
	}
	return dirs, nil
}

func (self *realSysFs) GetNetworkAddress(name string) (string, error) {
	address, err := ioutil.ReadFile(path.Join(netDir, name, "/address"))
	if err != nil {
		return "", err
	}
	return string(address), nil
}

func (self *realSysFs) GetNetworkMtu(name string) (string, error) {
	mtu, err := ioutil.ReadFile(path.Join(netDir, name, "/mtu"))
	if err != nil {
		return "", err
	}
	return string(mtu), nil
}

func (self *realSysFs) GetNetworkSpeed(name string) (string, error) {
	speed, err := ioutil.ReadFile(path.Join(netDir, name, "/speed"))
	if err != nil {
		return "", err
	}
	return string(speed), nil
}

func (self *realSysFs) GetNetworkStatValue(dev string, stat string) (uint64, error) {
	statPath := path.Join(netDir, dev, "/statistics", stat)
	out, err := ioutil.ReadFile(statPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read stat from %q for device %q", statPath, dev)
	}
	var s uint64
	n, err := fmt.Sscanf(string(out), "%d", &s)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("could not parse value from %q for file %s", string(out), statPath)
	}
	return s, nil
}

func (self *realSysFs) GetCaches(id int) ([]os.FileInfo, error) {
	cpuPath := fmt.Sprintf("%s%d/cache", cacheDir, id)
	return ioutil.ReadDir(cpuPath)
}

func bitCount(i uint64) (count int) {
	for i != 0 {
		if i&1 == 1 {
			count++
		}
		i >>= 1
	}
	return
}

func getCpuCount(cache string) (count int, err error) {
	out, err := ioutil.ReadFile(path.Join(cache, "/shared_cpu_map"))
	if err != nil {
		return 0, err
	}
	masks := strings.Split(string(out), ",")
	for _, mask := range masks {
		// convert hex string to uint64
		m, err := strconv.ParseUint(strings.TrimSpace(mask), 16, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse cpu map %q: %v", string(out), err)
		}
		count += bitCount(m)
	}
	return
}

func (self *realSysFs) GetCacheInfo(id int, name string) (CacheInfo, error) {
	cachePath := fmt.Sprintf("%s%d/cache/%s", cacheDir, id, name)
	out, err := ioutil.ReadFile(path.Join(cachePath, "/size"))
	if err != nil {
		return CacheInfo{}, err
	}
	var size uint64
	n, err := fmt.Sscanf(string(out), "%dK", &size)
	if err != nil || n != 1 {
		return CacheInfo{}, err
	}
	// convert to bytes
	size = size * 1024
	out, err = ioutil.ReadFile(path.Join(cachePath, "/level"))
	if err != nil {
		return CacheInfo{}, err
	}
	var level int
	n, err = fmt.Sscanf(string(out), "%d", &level)
	if err != nil || n != 1 {
		return CacheInfo{}, err
	}

	out, err = ioutil.ReadFile(path.Join(cachePath, "/type"))
	if err != nil {
		return CacheInfo{}, err
	}
	cacheType := strings.TrimSpace(string(out))
	cpuCount, err := getCpuCount(cachePath)
	if err != nil {
		return CacheInfo{}, err
	}
	return CacheInfo{
		Size:  size,
		Level: level,
		Type:  cacheType,
		Cpus:  cpuCount,
	}, nil
}

func (self *realSysFs) GetSystemUUID() (string, error) {
	id, err := ioutil.ReadFile(path.Join(dmiDir, "id", "product_uuid"))
	if err != nil {
		//If running on baremetal Power then UID is /proc/device-tree/system-id
		id, err = ioutil.ReadFile(path.Join(ppcDevTree, "system-id"))
		if err != nil {
			//If running on a KVM guest on Power then UUID is /proc/device-tree/vm,uuid
			id, err = ioutil.ReadFile(path.Join(ppcDevTree, "vm,uuid"))
			if err != nil {
				return "", err
			}
		}
	}
	return strings.TrimSpace(string(id)), nil
}

type InterfaceStats struct {
	// The name of the interface.
	Name string `json:"name"`
	// Cumulative count of bytes received.
	RxBytes uint64 `json:"rx_bytes"`
	// Cumulative count of packets received.
	RxPackets uint64 `json:"rx_packets"`
	// Cumulative count of receive errors encountered.
	RxErrors uint64 `json:"rx_errors"`
	// Cumulative count of packets dropped while receiving.
	RxDropped uint64 `json:"rx_dropped"`
	// Cumulative count of bytes transmitted.
	TxBytes uint64 `json:"tx_bytes"`
	// Cumulative count of packets transmitted.
	TxPackets uint64 `json:"tx_packets"`
	// Cumulative count of transmit errors encountered.
	TxErrors uint64 `json:"tx_errors"`
	// Cumulative count of packets dropped while transmitting.
	TxDropped uint64 `json:"tx_dropped"`
}

func GetNetworkStats(name string, sysFs SysFs) (InterfaceStats, error) {
	var stats InterfaceStats
	var err error
	stats.Name = name
	stats.RxBytes, err = sysFs.GetNetworkStatValue(name, "rx_bytes")
	if err != nil {
		return stats, err
	}
	stats.RxPackets, err = sysFs.GetNetworkStatValue(name, "rx_packets")
	if err != nil {
		return stats, err
	}
	stats.RxErrors, err = sysFs.GetNetworkStatValue(name, "rx_errors")
	if err != nil {
		return stats, err
	}
	stats.RxDropped, err = sysFs.GetNetworkStatValue(name, "rx_dropped")
	if err != nil {
		return stats, err
	}
	stats.TxBytes, err = sysFs.GetNetworkStatValue(name, "tx_bytes")
	if err != nil {
		return stats, err
	}
	stats.TxPackets, err = sysFs.GetNetworkStatValue(name, "tx_packets")
	if err != nil {
		return stats, err
	}
	stats.TxErrors, err = sysFs.GetNetworkStatValue(name, "tx_errors")
	if err != nil {
		return stats, err
	}
	stats.TxDropped, err = sysFs.GetNetworkStatValue(name, "tx_dropped")
	if err != nil {
		return stats, err
	}
	return stats, nil
}

type NetInfo struct {
	// Device name
	Name string `json:"name"`

	// Mac Address
	MacAddress string `json:"mac_address"`

	// Speed in MBits/s
	Speed int64 `json:"speed"`

	// Maximum Transmission Unit
	Mtu int64 `json:"mtu"`
}

func GetNetworkInfo(sysFs SysFs) ([]NetInfo, error) {
	devs, err := sysFs.GetNetworkDevices()
	if err != nil {
		return nil, err
	}
	netDevices := []NetInfo{}
	for _, dev := range devs {
		name := dev.Name()
		// Ignore docker, loopback, and veth devices.
		ignoredDevices := []string{"lo", "veth", "docker"}
		ignored := false
		for _, prefix := range ignoredDevices {
			if strings.HasPrefix(name, prefix) {
				ignored = true
				break
			}
		}
		if ignored {
			continue
		}
		address, err := sysFs.GetNetworkAddress(name)
		if err != nil {
			return nil, err
		}
		mtuStr, err := sysFs.GetNetworkMtu(name)
		if err != nil {
			return nil, err
		}
		var mtu int64
		n, err := fmt.Sscanf(mtuStr, "%d", &mtu)
		if err != nil || n != 1 {
			return nil, fmt.Errorf("could not parse mtu from %s for device %s", mtuStr, name)
		}
		netInfo := NetInfo{
			Name:       name,
			MacAddress: strings.TrimSpace(address),
			Mtu:        mtu,
		}
		speed, err := sysFs.GetNetworkSpeed(name)
		// Some devices don't set speed.
		if err == nil {
			var s int64
			n, err := fmt.Sscanf(speed, "%d", &s)
			if err != nil || n != 1 {
				return nil, fmt.Errorf("could not parse speed from %s for device %s", speed, name)
			}
			netInfo.Speed = s
		}
		netDevices = append(netDevices, netInfo)
	}
	return netDevices, nil
}
