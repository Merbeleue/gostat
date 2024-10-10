package main

import (
	"fmt"
	"gostat/service"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

const (
	updateInterval = time.Second
	maxDataPoints  = 60
)

// NetworkData 存储网络数据
type NetworkData struct {
	TotalRecv uint64
	TotalSent uint64
}

// 初始化屏幕并启动主循环
func main() {
	s, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err := s.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer s.Fini()

	defStyle := tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.ColorWhite)
	s.SetStyle(defStyle)

	netData := &NetworkData{}

	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-quit:
				return
			default:
				drawLayout(s, netData)
				s.Show()
				time.Sleep(updateInterval)
			}
		}
	}()

	for {
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				close(quit)
				return
			}
		}
	}
}

func drawLayout(s tcell.Screen, netData *NetworkData) {
	width, height := s.Size()
	s.Clear()

	drawBorder(s, 0, 0, width-1, height-1)
	drawTitle(s, width)

	// 左侧
	leftWidth := width / 2
	drawBox(s, 1, 2, leftWidth-2, height/3, "System Info")
	drawSystemInfo(s, 2, 3, leftWidth-4, height/3-2)

	drawBox(s, 1, height/3+2, leftWidth-2, height/3-1, "Memory Usage")
	drawMemoryUsage(s, 2, height/3+3, leftWidth-4, height/3-3)

	drawBox(s, 1, 2*height/3+1, leftWidth-2, height/3-2, "Docker Info")
	drawDockerInfo(s, 2, 2*height/3+2, leftWidth-4, height/3-4)

	// 右侧
	rightStart := leftWidth - 1
	drawBox(s, rightStart, 2, width-rightStart-1, height/3, "CPU & Load")
	drawCPUInfo(s, rightStart+1, 3, width-rightStart-3, height/3-2)

	drawBox(s, rightStart, height/3+2, width-rightStart-1, height/3-1, "Disk Usage")
	drawDiskUsage(s, rightStart+1, height/3+3, width-rightStart-3, height/3-3)

	drawBox(s, rightStart, 2*height/3+1, width-rightStart-1, height/3-2, "Network Traffic")
	drawNetworkTraffic(s, rightStart+1, 2*height/3+2, width-rightStart-3, height/3-4, netData)
}

// drawTitle 函数在屏幕顶部绘制标题
func drawTitle(s tcell.Screen, width int) {
	title := "GoStat - System Monitor"
	style := tcell.StyleDefault.Foreground(tcell.ColorGreen).Bold(true)
	drawCenteredText(s, 0, 1, width, title, style)
}

// drawSystemInfo 函数显示系统信息
func drawSystemInfo(s tcell.Screen, x, y, width, height int) {
	osinfo, _ := service.GetOSInfo()
	info := fmt.Sprintf(
		"OS Version: %s\nHostName: %s\nUptime: %s\nCPU Info: %s",
		osinfo.OS, osinfo.Hostname, osinfo.Uptime, osinfo.Cpu,
	)
	addCenteredContent(s, x, y, width, height, info)
}

// drawMemoryUsage 函数显示内存使用情况
func drawMemoryUsage(s tcell.Screen, x, y, width, height int) {
	meminfo, _ := service.GetMemoryUse()

	// 计算垂直中心
	centerY := y + height/2 - 2

	drawBar(s, x, centerY, width, 100, 100, tcell.ColorGreen)
	drawCenteredText(s, x, centerY+1, width, fmt.Sprintf("Total: %dG", meminfo.MemoryTotal), tcell.StyleDefault.Foreground(tcell.ColorWhite))

	usedPercent := float64(meminfo.MemoryUsed) / float64(meminfo.MemoryTotal) * 100
	drawBar(s, x, centerY+3, width, usedPercent, 100, tcell.ColorOrange)
	drawCenteredText(s, x, centerY+4, width, fmt.Sprintf("Used: %dG (%.2f%%)", meminfo.MemoryUsed, usedPercent), tcell.StyleDefault.Foreground(tcell.ColorOrange))
}

// drawDockerInfo 函数显示Docker相关信息
func drawDockerInfo(s tcell.Screen, x, y, width, height int) {
	dockerInfo, err := service.GetDockerInfo()
	if err != nil || dockerInfo == nil {
		// 如果未安装 Docker，则显示提示信息
		drawCenteredText(s, x, y, width, "Docker is not installed or not running.", tcell.StyleDefault.Foreground(tcell.ColorRed))
		return
	}

	// 计算列宽
	colWidth := width / 2

	// 左列：基本信息
	leftInfo := fmt.Sprintf(
		"Running Containers: %d\nStopped Containers: %d\nTotal Images: %d",
		dockerInfo.RunningContainers,
		dockerInfo.StoppedContainers,
		dockerInfo.TotalImages,
	)
	addContentVerticalAlign(s, x, y, colWidth, height, leftInfo, tcell.StyleDefault)

	// 右列：最近的容器
	rightX := x + colWidth
	drawText(s, rightX, y, colWidth, 1, "Recent Containers:", tcell.StyleDefault.Foreground(tcell.ColorYellow))

	for i, container := range dockerInfo.RecentContainers {
		if i >= height-1 {
			break
		}
		// 如果容器名称太长，则截断
		containerInfo := fmt.Sprintf("- %s", container)
		if len(containerInfo) > colWidth {
			containerInfo = containerInfo[:colWidth-3] + "..."
		}
		drawText(s, rightX, y+1+i, colWidth, 1, containerInfo, tcell.StyleDefault.Foreground(tcell.ColorDarkCyan))
	}
}

// drawCPUInfo 函数显示CPU使用率和负载信息
func drawCPUInfo(s tcell.Screen, x, y, width, height int) {
	cpuUsage, _ := service.GetCPUUsage()
	loadAvg, _ := service.GetLoadAverage()

	// 计算垂直中心
	centerY := y + height/2 - 2

	drawCPUGraph(s, x, centerY, width, 1, cpuUsage)
	drawCenteredText(s, x, centerY+2, width, fmt.Sprintf("CPU Usage: %.2f%%", cpuUsage), tcell.StyleDefault.Foreground(tcell.ColorWhite))

	loadInfo := "Load Average:"
	drawCenteredText(s, x, centerY+4, width, loadInfo, tcell.StyleDefault.Foreground(tcell.ColorYellow))
	loadDetails := fmt.Sprintf("1min: %.2f  5min: %.2f  15min: %.2f", loadAvg.Load1, loadAvg.Load5, loadAvg.Load15)
	drawCenteredText(s, x, centerY+5, width, loadDetails, tcell.StyleDefault.Foreground(tcell.ColorYellow))
}

// drawDiskUsage 函数显示磁盘使用情况
func drawDiskUsage(s tcell.Screen, x, y, width, height int) {
	diskinfo, _ := service.GetDiskUse()
	info := fmt.Sprintf(
		"Total: %dG Used: %dG Total Usage: %.2f%%\n%s\n%s\n%s     %s       %dG     %.2f%%",
		diskinfo.DiskTotal, diskinfo.DiskUsed, diskinfo.DiskPercent,
		strings.Repeat("=", width),
		"Disk       Mounted    Free    Used",
		diskinfo.DiskRootDevice, diskinfo.DiskRootMount, diskinfo.DiskRootFree, diskinfo.DiskRootPercent,
	)
	addCenteredContent(s, x, y, width, height, info)
}

// drawNetworkTraffic 函数显示网络流量信息
func drawNetworkTraffic(s tcell.Screen, x, y, width, height int, netData *NetworkData) {
	netinfo, _ := service.GetNetworkUsage()
	updateNetworkData(netData, netinfo)

	maxTraffic := uint64(1024 * 1024 * 1024)

	// 计算垂直中心
	centerY := y + height/2 - 2

	recvPercent := float64(netData.TotalRecv) / float64(maxTraffic) * 100
	drawBar(s, x, centerY, width, recvPercent, 100, tcell.ColorGreen)
	drawCenteredText(s, x, centerY+1, width, fmt.Sprintf("Recv: %.2f MB", float64(netData.TotalRecv)/1024/1024), tcell.StyleDefault.Foreground(tcell.ColorGreen))

	sentPercent := float64(netData.TotalSent) / float64(maxTraffic) * 100
	drawBar(s, x, centerY+3, width, sentPercent, 100, tcell.ColorRed)
	drawCenteredText(s, x, centerY+4, width, fmt.Sprintf("Sent: %.2f MB", float64(netData.TotalSent)/1024/1024), tcell.StyleDefault.Foreground(tcell.ColorRed))
}

// drawCPUGraph 函数绘制CPU使用率图形
func drawCPUGraph(s tcell.Screen, x, y, width, height int, cpuUsage float64) {
	usedWidth := int(float64(width) * cpuUsage / 100)
	for i := 0; i < width; i++ {
		if i < usedWidth {
			s.SetContent(x+i, y, '█', nil, tcell.StyleDefault.Foreground(tcell.ColorRed))
		} else {
			s.SetContent(x+i, y, '░', nil, tcell.StyleDefault.Foreground(tcell.ColorGray))
		}
	}
}

// drawBar 函数绘制一个进度条
func drawBar(s tcell.Screen, x, y, width int, value, max float64, color tcell.Color) {
	filledWidth := int(float64(width) * value / max)
	for i := 0; i < width; i++ {
		if i < filledWidth {
			s.SetContent(x+i, y, '█', nil, tcell.StyleDefault.Foreground(color))
		} else {
			s.SetContent(x+i, y, '░', nil, tcell.StyleDefault.Foreground(tcell.ColorGray))
		}
	}
}

// updateNetworkData 函数更新网络数据
func updateNetworkData(data *NetworkData, netinfo *service.HostsInfo) {
	data.TotalRecv = netinfo.TotalBytesRecv
	data.TotalSent = netinfo.TotalBytesSent
}

// drawBorder 函数绘制边框
func drawBorder(s tcell.Screen, x1, y1, x2, y2 int) {
	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorLightCoral)

	// Top and bottom borders
	for x := x1; x <= x2; x++ {
		s.SetContent(x, y1, '-', nil, borderStyle)
		s.SetContent(x, y2, '-', nil, borderStyle)
	}

	// Left and right borders
	for y := y1; y <= y2; y++ {
		s.SetContent(x1, y, '|', nil, borderStyle)
		s.SetContent(x2, y, '|', nil, borderStyle)
	}

	// Corners
	s.SetContent(x1, y1, '+', nil, borderStyle)
	s.SetContent(x2, y1, '+', nil, borderStyle)
	s.SetContent(x1, y2, '+', nil, borderStyle)
	s.SetContent(x2, y2, '+', nil, borderStyle)
}

// drawBox 函数绘制一个带标题的框
func drawBox(s tcell.Screen, x, y, width, height int, title string) {
	drawBorder(s, x, y, x+width-1, y+height-1)

	// Draw title
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
	titleX := x + (width-len(title))/2
	drawText(s, titleX, y, width, 1, title, titleStyle)
}

// addCenteredContent 函数在指定区域内居中添加内容
func addCenteredContent(s tcell.Screen, x, y, width, height int, content string) {
	lines := strings.Split(content, "\n")
	startY := y + (height-len(lines))/2
	for i, line := range lines {
		if i >= height {
			break
		}
		drawCenteredText(s, x, startY+i, width, line, tcell.StyleDefault)
	}
}

// drawCenteredText 函数在指定位置绘制居中文本
func drawCenteredText(s tcell.Screen, x, y, width int, text string, style tcell.Style) {
	textWidth := len([]rune(text))
	startX := x + (width-textWidth)/2
	for i, r := range []rune(text) {
		if i >= width {
			break
		}
		s.SetContent(startX+i, y, r, nil, style)
	}
}

// drawText 函数在指定位置绘制左对齐文本
func drawText(s tcell.Screen, x, y, width, height int, text string, style tcell.Style) {
	for i, r := range []rune(text) {
		if i >= width {
			break
		}
		s.SetContent(x+i, y, r, nil, style)
	}
}

// addContentVerticalAlign 函数添加垂直对齐的内容
func addContentVerticalAlign(s tcell.Screen, x, y, width, height int, content string, style tcell.Style) {
	lines := strings.Split(content, "\n")
	startY := y + (height-len(lines))/2
	for i, line := range lines {
		if i >= height {
			break
		}
		drawText(s, x, startY+i, width, 1, line, style)
	}
}
