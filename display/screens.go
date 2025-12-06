package display

import (
	"fmt"
	"github.com/shirou/gopsutil/v4/mem"
	"image"
	"image/draw"
	"os"
	"strings"
	"time"

	"cloudkey/images"
	"cloudkey/src/network"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

func buildNetwork(i int, demo bool) {
	screen := screens[i]
	hostname := "Simons cloudkey"
	lan := "192.168.11.13"
	wan := "203.0.113.32"

	draw.Draw(screen, screen.Bounds(), image.Black, image.ZP, draw.Src)
	draw.Draw(screen, image.Rect(2, 2, 2+16, 2+16), images.Load("host"), image.ZP, draw.Src)
	draw.Draw(screen, image.Rect(2, 22, 2+16, 22+16), images.Load("network"), image.ZP, draw.Src)
	draw.Draw(screen, image.Rect(2, 42, 2+16, 42+16), images.Load("internet"), image.ZP, draw.Src)

	// Loop Every Hour
	go func() {
		for {
			if !demo {
				hostname, _ = os.Hostname()
			}
			write(screen, hostname, 22, 1, 12, "lato-regular")

			if !demo {
				lan, _ = network.LANIP()
			}
			write(screen, lan, 22, 21, 12, "lato-regular")

			if !demo {
				wan, _ = network.WANIP()
			}
			write(screen, wan, 22, 41, 12, "lato-regular")

			time.Sleep(59 * time.Minute)
		}
	}()
}

func buildSpeedTest(i int, demo bool, opts CmdLineOpts) {
	dmsg := "fetching..."
	umsg := "fetching..."
	tmsg := "from UDM Pro"

	screen := screens[i]

	draw.Draw(screen, screen.Bounds(), image.Black, image.ZP, draw.Src)
	draw.Draw(screen, image.Rect(2, 2, 2+16, 2+16), images.Load("download"), image.ZP, draw.Src)
	draw.Draw(screen, image.Rect(2, 22, 2+16, 22+16), images.Load("upload"), image.ZP, draw.Src)
	draw.Draw(screen, image.Rect(2, 42, 2+16, 42+16), images.Load("clock"), image.ZP, draw.Src)

	if demo {
		dmsg = "1.2 Gb/s" // Show Gbps example in demo
		umsg = "43.9 Mb/s"
		tmsg = "25 minutes ago"
		write(screen, dmsg, 22, 1, 12, "lato-regular")
		write(screen, umsg, 22, 21, 12, "lato-regular")
		write(screen, tmsg, 22, 41, 12, "lato-regular")
	} else {
		// Smart speedtest fetching - check for new results every 5 minutes
		go func() {
			var lastResult *network.SpeedtestResult
			var lastFetchTime time.Time
			var lastKnownTimestamp int64

			// Initial fetch immediately at startup
			fmt.Println("Fetching initial speedtest data immediately...")

			for {
				now := time.Now()

				// Always check every 5 minutes, but respect minimum interval
				shouldFetch := false

				if lastResult == nil {
					shouldFetch = true
					fmt.Println("No cached speedtest data - fetching initial data")
				} else if time.Since(lastFetchTime) >= 5*time.Minute {
					shouldFetch = true
					fmt.Printf("5 minutes elapsed - checking for new speedtest results\n")
				}

				if shouldFetch {
					result, err := network.GetUDMProSpeedtest(
						opts.UDMBaseURL,
						opts.UDMUsername,
						opts.UDMPassword,
						opts.UDMSite,
						opts.UDMVersion,
					)
					if err != nil {
						fmt.Printf("Error fetching UDM Pro speedtest: %v\n", err)
						if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "cannot reach") {
							dmsg = "network error"
							umsg = "check UDM IP"
							tmsg = "verify connectivity"
						} else if strings.Contains(err.Error(), "connection refused") {
							dmsg = "UDM offline"
							umsg = "check device"
							tmsg = "verify running"
						} else if strings.Contains(err.Error(), "login failed") || strings.Contains(err.Error(), "403") {
							dmsg = "auth error"
							umsg = "403 forbidden"
							tmsg = "check credentials"
						} else if strings.Contains(err.Error(), "429") {
							dmsg = "rate limited"
							umsg = "retry tomorrow"
							tmsg = "API limit hit"
						} else {
							dmsg = "connection error"
							umsg = "check logs"
							tmsg = "see UDM_SETUP"
						}
					} else {
						// Success - check if this is newer data
						isNewer := lastKnownTimestamp == 0 || result.Timestamp > lastKnownTimestamp

						if isNewer {
							fmt.Printf("Found newer speedtest data (timestamp: %d)\n", result.Timestamp)
							lastResult = result
							lastKnownTimestamp = result.Timestamp
							dmsg = network.FormatSpeed(result.DownloadMbps)
							umsg = network.FormatSpeed(result.UploadMbps)
							tmsg = network.GetRelativeTime(result.Timestamp)
							fmt.Printf("UDM Pro Speedtest - Download: %.1f Mb/s, Upload: %.1f Mb/s, Latency: %.1f ms\n",
								result.DownloadMbps, result.UploadMbps, result.LatencyMs)
						} else {
							fmt.Printf("No new speedtest data (still timestamp: %d)\n", lastKnownTimestamp)
						}

						// Always update fetch time regardless of whether data is new
						lastFetchTime = now
					}
				} else {
					// Use cached data
					if lastResult != nil {
						dmsg = network.FormatSpeed(lastResult.DownloadMbps)
						umsg = network.FormatSpeed(lastResult.UploadMbps)
						tmsg = network.GetRelativeTime(lastResult.Timestamp)
					} else {
						// No data yet, show waiting message
						cst := now.Add(-6 * time.Hour)
						if cst.Hour() < 14 {
							dmsg = "waiting"
							umsg = "test at 2pm"
							tmsg = "CST today"
						} else {
							dmsg = "no test yet"
							umsg = "check after"
							tmsg = "2pm CST"
						}
					}
				}

				// Clear and redraw the screen
				draw.Draw(screen, image.Rect(20, 0, 160, 60), image.Black, image.ZP, draw.Src)
				write(screen, dmsg, 22, 1, 12, "lato-regular")
				write(screen, umsg, 22, 21, 12, "lato-regular")
				write(screen, tmsg, 22, 41, 12, "lato-regular")

				// Check for updates every 5 minutes
				time.Sleep(5 * time.Minute)
			}
		}()
	}
}

func buildSystemStats(i int, demo bool) {

	screen := screens[i]

	// Loop to update stats periodically
	go func() {
		for {
			v, _ := mem.VirtualMemory()
			used := float64(v.Used) / (1024 * 1024 * 1024)
			total := float64(v.Total) / (1024 * 1024 * 1024)
			usedPercent := v.UsedPercent

			ramInfo := fmt.Sprintf(" %.1f/%.1fGB %.1f%%", used, total, usedPercent)

			cpuUsage, _ := getCPUUsagePerCore()
			cpuInfo := fmt.Sprintf(" %.1f%%", cpuUsage)

			// fmt.Println("Used:", used)
			// fmt.Println("Total:", total)
			// fmt.Println("CPU Usage:", cpuInfo)

			// Clear the screen
			draw.Draw(screen, screen.Bounds(), image.Black, image.ZP, draw.Src)

			// Draw static labels for CPU and RAM
			draw.Draw(screen, image.Rect(2, 2, 2+16, 22+16), images.Load("ram"), image.ZP, draw.Src)
			draw.Draw(screen, image.Rect(2, 22, 2+16, 22+16), images.Load("cpu"), image.ZP, draw.Src)

			// Clear the screen
			write(screen, ramInfo, 22, 1, 12, "lato-regular")
			write(screen, cpuInfo, 22, 21, 12, "lato-regular")

			time.Sleep(5 * time.Second)
		}
	}()
}

func getCPUUsagePerCore() (float64, error) {
	// Read CPU stats
	stat, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		return 0, err
	}

	// Loop through all cores and calculate the usage
	var totalCPUUsage uint64
	var totalCPUTime uint64
	for _, stats := range stat.CPUStats {
		// Extract stats for each core
		user := stats.User
		system := stats.System
		idle := stats.Idle
		IOWait := stats.IOWait

		// Calculate total time spent (user + system + idle + IOWait)
		total := user + system + idle + IOWait

		// Calculate the total active time (user + system + IOWait)
		active := user + system + IOWait

		// Accumulate totals
		totalCPUUsage += active
		totalCPUTime += total
	}

	// Calculate the total CPU usage as a percentage
	if totalCPUTime == 0 {
		return 0, nil // Avoid division by zero
	}

	usagePercentage := (float64(totalCPUUsage) / float64(totalCPUTime)) * 100
	return usagePercentage, nil
}
