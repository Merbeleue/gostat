package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"golang.org/x/net/context"
)

// HostsInfo 结构体存储系统信息
type HostsInfo struct {
	OS                string
	Hostname          string
	Uptime            time.Duration
	Cpu               string
	CpuUsage          float64
	DiskRootDevice    string
	DiskRootMount     string
	DiskRootFree      uint64
	DiskRootTotal     uint64
	DiskRootUsed      uint64
	DiskRootPercent   float64
	DiskTotal         uint64
	DiskUsed          uint64
	DiskPercent       float64
	MemoryTotal       uint64
	MemoryUsed        uint64
	MemoryFree        uint64
	MemoryCached      uint64
	MemoryBuffers     uint64
	MemoryMainPercent float64
	MemorySwapPercent float64
	TotalBytesRecv    uint64
	TotalBytesSent    uint64
	RecentBytesRecv   uint64
	RecentBytesSent   uint64
}

// LoadAverage 结构体存储系统负载信息
type LoadAverage struct {
	Load1  float64
	Load5  float64
	Load15 float64
}

// DockerInfo 结构体存储 Docker 相关信息
type DockerInfo struct {
	RunningContainers int
	StoppedContainers int
	TotalImages       int
	RecentContainers  []string
}

var (
	lastBytesRecv uint64
	lastBytesSent uint64
	lastCheckTime time.Time
	netMutex      sync.Mutex
)

// GetOSInfo 获取操作系统信息
func GetOSInfo() (*HostsInfo, error) {
	hostinfo := &HostsInfo{}

	osversion, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("获取系统版本失败: %v", err)
	}
	hostinfo.OS = fmt.Sprintf("%s %s", osversion.Platform, osversion.PlatformVersion)
	hostinfo.Hostname = osversion.Hostname

	uptimeinfo, err := host.Uptime()
	if err != nil {
		return nil, fmt.Errorf("获取系统运行时间失败: %v", err)
	}
	hostinfo.Uptime = time.Duration(uptimeinfo) * time.Second

	cpuinfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("获取CPU信息失败: %v", err)
	}
	if len(cpuinfo) > 0 {
		hostinfo.Cpu = cpuinfo[0].ModelName
	}

	cpuUsage, err := GetCPUUsage()
	if err != nil {
		return nil, fmt.Errorf("获取CPU使用率失败: %v", err)
	}
	hostinfo.CpuUsage = cpuUsage

	return hostinfo, nil
}

// GetDiskUse 获取磁盘使用情况
func GetDiskUse() (*HostsInfo, error) {
	diskinfo := &HostsInfo{}

	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, fmt.Errorf("获取分区信息失败: %v", err)
	}

	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			return nil, fmt.Errorf("获取分区 %s 的使用情况失败: %v", partition.Mountpoint, err)
		}
		diskinfo.DiskTotal += (usage.Total) / 1024 / 1024 / 1024
		diskinfo.DiskUsed += (usage.Used) / 1024 / 1024 / 1024

		if partition.Mountpoint == "/" {
			diskinfo.DiskRootDevice = partition.Device
			diskinfo.DiskRootMount = partition.Mountpoint
			diskinfo.DiskRootFree = (usage.Free) / 1024 / 1024 / 1024
			diskinfo.DiskRootTotal = (usage.Total) / 1024 / 1024 / 1024
			diskinfo.DiskRootUsed = (usage.Used) / 1024 / 1024 / 1024
			diskinfo.DiskRootPercent = usage.UsedPercent
		}
	}

	if diskinfo.DiskTotal > 0 {
		diskinfo.DiskPercent = float64(diskinfo.DiskUsed) / float64(diskinfo.DiskTotal) * 100
	}

	return diskinfo, nil
}

// GetMemoryUse 获取内存使用情况
func GetMemoryUse() (*HostsInfo, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("获取内存信息失败: %v", err)
	}

	swapInfo, err := mem.SwapMemory()
	if err != nil {
		return nil, fmt.Errorf("获取交换内存信息失败: %v", err)
	}

	return &HostsInfo{
		MemoryTotal:       memInfo.Total / 1024 / 1024 / 1024,
		MemoryUsed:        memInfo.Used / 1024 / 1024 / 1024,
		MemoryFree:        memInfo.Free / 1024 / 1024 / 1024,
		MemoryCached:      memInfo.Cached / 1024 / 1024 / 1024,
		MemoryBuffers:     memInfo.Buffers / 1024 / 1024 / 1024,
		MemoryMainPercent: memInfo.UsedPercent,
		MemorySwapPercent: swapInfo.UsedPercent,
	}, nil
}

// GetNetworkUsage 获取网络使用情况
func GetNetworkUsage() (*HostsInfo, error) {
	netInfo := &HostsInfo{}

	interfaces, err := net.IOCounters(false)
	if err != nil {
		return nil, fmt.Errorf("获取网络流量信息失败: %v", err)
	}

	var totalBytesRecv, totalBytesSent uint64
	if len(interfaces) > 0 {
		totalBytesRecv = interfaces[0].BytesRecv
		totalBytesSent = interfaces[0].BytesSent
	}

	netMutex.Lock()
	defer netMutex.Unlock()

	now := time.Now()
	duration := now.Sub(lastCheckTime).Seconds()

	if duration > 0 && lastCheckTime.Unix() != 0 {
		netInfo.RecentBytesRecv = uint64(float64(totalBytesRecv-lastBytesRecv) / duration)
		netInfo.RecentBytesSent = uint64(float64(totalBytesSent-lastBytesSent) / duration)
	}

	netInfo.TotalBytesRecv = totalBytesRecv
	netInfo.TotalBytesSent = totalBytesSent

	lastBytesRecv = totalBytesRecv
	lastBytesSent = totalBytesSent
	lastCheckTime = now

	return netInfo, nil
}

// GetCPUUsage 获取CPU使用率
func GetCPUUsage() (float64, error) {
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, fmt.Errorf("获取CPU使用率失败: %v", err)
	}
	if len(percent) > 0 {
		return percent[0], nil
	}
	return 0, nil
}

// GetLoadAverage 获取系统负载信息
func GetLoadAverage() (*LoadAverage, error) {
	loadInfo, err := load.Avg()
	if err != nil {
		return nil, fmt.Errorf("获取系统负载信息失败: %v", err)
	}

	return &LoadAverage{
		Load1:  loadInfo.Load1,
		Load5:  loadInfo.Load5,
		Load15: loadInfo.Load15,
	}, nil
}

// GetDockerInfo 获取Docker信息
func GetDockerInfo() (*DockerInfo, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("创建Docker客户端失败: %v", err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %v", err)
	}

	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取镜像列表失败: %v", err)
	}

	dockerInfo := &DockerInfo{
		TotalImages: len(images),
	}

	for _, container := range containers {
		if container.State == "running" {
			dockerInfo.RunningContainers++
		} else {
			dockerInfo.StoppedContainers++
		}
	}

	// 获取最近创建的容器
	for i, container := range containers {
		if i >= 3 {
			break
		}
		dockerInfo.RecentContainers = append(dockerInfo.RecentContainers, fmt.Sprintf("%s (%s)", container.Names[0], container.State))
	}

	return dockerInfo, nil
}
