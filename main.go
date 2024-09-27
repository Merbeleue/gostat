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

type NetworkData struct {
	TotalRecv uint64
	TotalSent uint64
}

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

	// Left side
	leftWidth := width / 2
	drawBox(s, 1, 2, leftWidth-2, height/3, "System Info")
	drawSystemInfo(s, 2, 3, leftWidth-4, height/3-2)

	drawBox(s, 1, height/3+2, leftWidth-2, height/3-1, "Memory Usage")
	drawMemoryUsage(s, 2, height/3+3, leftWidth-4, height/3-3)

	drawBox(s, 1, 2*height/3+1, leftWidth-2, height/3-2, "Docker Info")
	drawDockerInfo(s, 2, 2*height/3+2, leftWidth-4, height/3-4)

	// Right side
	rightStart := leftWidth - 1
	drawBox(s, rightStart, 2, width-rightStart-1, height/3, "CPU & Load")
	drawCPUInfo(s, rightStart+1, 3, width-rightStart-3, height/3-2)

	drawBox(s, rightStart, height/3+2, width-rightStart-1, height/3-1, "Disk Usage")
	drawDiskUsage(s, rightStart+1, height/3+3, width-rightStart-3, height/3-3)

	drawBox(s, rightStart, 2*height/3+1, width-rightStart-1, height/3-2, "Network Traffic")
	drawNetworkTraffic(s, rightStart+1, 2*height/3+2, width-rightStart-3, height/3-4, netData)
}

func drawTitle(s tcell.Screen, width int) {
	title := "GoStat - System Monitor"
	style := tcell.StyleDefault.Foreground(tcell.ColorGreen).Bold(true)
	drawCenteredText(s, 0, 1, width, title, style)
}

func drawSystemInfo(s tcell.Screen, x, y, width, height int) {
	osinfo, _ := service.GetOSInfo()
	info := fmt.Sprintf(
		"OS Version: %s\nHostName: %s\nUptime: %s\nCPU Info: %s",
		osinfo.OS, osinfo.Hostname, osinfo.Uptime, osinfo.Cpu,
	)
	addCenteredContent(s, x, y, width, height, info)
}

func drawMemoryUsage(s tcell.Screen, x, y, width, height int) {
	meminfo, _ := service.GetMemoryUse()

	// Calculate vertical center
	centerY := y + height/2 - 2

	drawBar(s, x, centerY, width, 100, 100, tcell.ColorGreen)
	drawCenteredText(s, x, centerY+1, width, fmt.Sprintf("Total: %dG", meminfo.MemoryTotal), tcell.StyleDefault.Foreground(tcell.ColorWhite))

	usedPercent := float64(meminfo.MemoryUsed) / float64(meminfo.MemoryTotal) * 100
	drawBar(s, x, centerY+3, width, usedPercent, 100, tcell.ColorOrange)
	drawCenteredText(s, x, centerY+4, width, fmt.Sprintf("Used: %dG (%.2f%%)", meminfo.MemoryUsed, usedPercent), tcell.StyleDefault.Foreground(tcell.ColorOrange))
}

func drawDockerInfo(s tcell.Screen, x, y, width, height int) {
	dockerInfo, _ := service.GetDockerInfo()

	// Calculate column widths
	colWidth := width / 2

	// Left column: Basic info
	leftInfo := fmt.Sprintf(
		"Running Containers: %d\nStopped Containers: %d\nTotal Images: %d",
		dockerInfo.RunningContainers,
		dockerInfo.StoppedContainers,
		dockerInfo.TotalImages,
	)
	addContentVerticalAlign(s, x, y, colWidth, height, leftInfo, tcell.StyleDefault)

	// Right column: Recent Containers
	rightX := x + colWidth
	drawText(s, rightX, y, colWidth, 1, "Recent Containers:", tcell.StyleDefault.Foreground(tcell.ColorYellow))

	for i, container := range dockerInfo.RecentContainers {
		if i >= height-1 {
			break
		}
		// Truncate container name if it's too long
		containerInfo := fmt.Sprintf("- %s", container)
		if len(containerInfo) > colWidth {
			containerInfo = containerInfo[:colWidth-3] + "..."
		}
		drawText(s, rightX, y+1+i, colWidth, 1, containerInfo, tcell.StyleDefault.Foreground(tcell.ColorDarkCyan))
	}
}

func drawCPUInfo(s tcell.Screen, x, y, width, height int) {
	cpuUsage, _ := service.GetCPUUsage()
	loadAvg, _ := service.GetLoadAverage()

	// Calculate vertical center
	centerY := y + height/2 - 2

	drawCPUGraph(s, x, centerY, width, 1, cpuUsage)
	drawCenteredText(s, x, centerY+2, width, fmt.Sprintf("CPU Usage: %.2f%%", cpuUsage), tcell.StyleDefault.Foreground(tcell.ColorWhite))

	loadInfo := "Load Average:"
	drawCenteredText(s, x, centerY+4, width, loadInfo, tcell.StyleDefault.Foreground(tcell.ColorYellow))
	loadDetails := fmt.Sprintf("1min: %.2f  5min: %.2f  15min: %.2f", loadAvg.Load1, loadAvg.Load5, loadAvg.Load15)
	drawCenteredText(s, x, centerY+5, width, loadDetails, tcell.StyleDefault.Foreground(tcell.ColorYellow))
}

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

func drawNetworkTraffic(s tcell.Screen, x, y, width, height int, netData *NetworkData) {
	netinfo, _ := service.GetNetworkUsage()
	updateNetworkData(netData, netinfo)

	maxTraffic := uint64(1024 * 1024 * 1024) // 1GB as max

	// Calculate vertical center
	centerY := y + height/2 - 2

	recvPercent := float64(netData.TotalRecv) / float64(maxTraffic) * 100
	drawBar(s, x, centerY, width, recvPercent, 100, tcell.ColorGreen)
	drawCenteredText(s, x, centerY+1, width, fmt.Sprintf("Recv: %.2f MB", float64(netData.TotalRecv)/1024/1024), tcell.StyleDefault.Foreground(tcell.ColorGreen))

	sentPercent := float64(netData.TotalSent) / float64(maxTraffic) * 100
	drawBar(s, x, centerY+3, width, sentPercent, 100, tcell.ColorRed)
	drawCenteredText(s, x, centerY+4, width, fmt.Sprintf("Sent: %.2f MB", float64(netData.TotalSent)/1024/1024), tcell.StyleDefault.Foreground(tcell.ColorRed))
}

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

func updateNetworkData(data *NetworkData, netinfo *service.HostsInfo) {
	data.TotalRecv = netinfo.TotalBytesRecv
	data.TotalSent = netinfo.TotalBytesSent
}

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

func drawBox(s tcell.Screen, x, y, width, height int, title string) {
	drawBorder(s, x, y, x+width-1, y+height-1)

	// Draw title
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
	titleX := x + (width-len(title))/2
	drawText(s, titleX, y, width, 1, title, titleStyle)
}

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

// Add this new function to draw left-aligned text
func drawText(s tcell.Screen, x, y, width, height int, text string, style tcell.Style) {
	for i, r := range []rune(text) {
		if i >= width {
			break
		}
		s.SetContent(x+i, y, r, nil, style)
	}
}

// New function to add content with vertical alignment
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
