package handler

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsHandler handles Prometheus metrics endpoint
type MetricsHandler struct {
	startTime time.Time
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{
		startTime: time.Now(),
	}
}

// GetMetrics returns Prometheus-compatible metrics
func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(h.startTime).Seconds()

	// Prometheus text format
	metrics := `# HELP sub2api_uptime_seconds Server uptime in seconds
# TYPE sub2api_uptime_seconds gauge
sub2api_uptime_seconds ` + formatFloat(uptime) + `

# HELP sub2api_memory_alloc_bytes Memory allocated in bytes
# TYPE sub2api_memory_alloc_bytes gauge
sub2api_memory_alloc_bytes ` + formatUint64(m.Alloc) + `

# HELP sub2api_memory_sys_bytes Total memory obtained from OS in bytes
# TYPE sub2api_memory_sys_bytes gauge
sub2api_memory_sys_bytes ` + formatUint64(m.Sys) + `

# HELP sub2api_goroutines Number of goroutines
# TYPE sub2api_goroutines gauge
sub2api_goroutines ` + formatInt(runtime.NumGoroutine()) + `

# HELP sub2api_gc_runs_total Total number of GC runs
# TYPE sub2api_gc_runs_total counter
sub2api_gc_runs_total ` + formatUint32(m.NumGC) + `
`

	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(metrics))
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func formatUint64(u uint64) string {
	return fmt.Sprintf("%d", u)
}

func formatUint32(u uint32) string {
	return fmt.Sprintf("%d", u)
}

func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}
