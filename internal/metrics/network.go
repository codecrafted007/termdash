package metrics

import (
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/host"
	psnet "github.com/shirou/gopsutil/v3/net"
)

func (c *Collector) collectNetwork(now time.Time) (NetworkMetrics, error) {
	counters, err := psnet.IOCounters(false)
	if err != nil {
		return NetworkMetrics{}, err
	}
	if len(counters) == 0 {
		return NetworkMetrics{}, nil
	}

	bytesSent := counters[0].BytesSent
	bytesRecv := counters[0].BytesRecv

	var sendRate, recvRate float64
	if c.initialized {
		elapsed := now.Sub(c.prevTime).Seconds()
		if elapsed > 0 {
			sendRate = float64(bytesSent-c.prevBytesSent) / elapsed
			recvRate = float64(bytesRecv-c.prevBytesRecv) / elapsed
		}
	}

	c.prevBytesSent = bytesSent
	c.prevBytesRecv = bytesRecv
	c.prevTime = now
	c.initialized = true

	return NetworkMetrics{
		BytesSent: bytesSent,
		BytesRecv: bytesRecv,
		SendRate:  sendRate,
		RecvRate:  recvRate,
	}, nil
}

func collectHostInfo() (hostname string, osInfo string, uptime time.Duration) {
	info, err := host.Info()
	if err != nil {
		return "unknown", runtime.GOOS, 0
	}

	hostname = info.Hostname
	osInfo = fmt.Sprintf("%s %s", info.Platform, info.PlatformVersion)
	if osInfo == " " {
		osInfo = runtime.GOOS
	}
	uptime = time.Duration(info.Uptime) * time.Second
	return
}
