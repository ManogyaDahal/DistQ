package config

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// MemoryDetails describes how much memory the worker process can spend on its
// pool and the estimated memory cost of one worker slot.
type MemoryDetails struct {
	TotalMemoryMB     int `json:"total_memory_mb"`
	AvailableMemoryMB int `json:"available_memory_mb"`
	MemoryPerWorkerMB int `json:"memory_per_worker_mb"`
}

func LoadMemoryDetails(path string) (*MemoryDetails, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read memory details: %w", err)
	}

	var details MemoryDetails
	if err := json.Unmarshal(data, &details); err != nil {
		return nil, fmt.Errorf("parse memory details: %w", err)
	}

	if details.TotalMemoryMB < 1 {
		return nil, fmt.Errorf("memory details: total_memory_mb must be at least 1")
	}
	if details.AvailableMemoryMB < 1 {
		return nil, fmt.Errorf("memory details: available_memory_mb must be at least 1")
	}
	if details.MemoryPerWorkerMB < 1 {
		return nil, fmt.Errorf("memory details: memory_per_worker_mb must be at least 1")
	}
	if details.AvailableMemoryMB > details.TotalMemoryMB {
		return nil, fmt.Errorf("memory details: available_memory_mb cannot exceed total_memory_mb")
	}

	return &details, nil
}

func LoadDeviceMemoryDetails(memoryPerWorkerMB int) (*MemoryDetails, error) {
	if memoryPerWorkerMB < 1 {
		return nil, fmt.Errorf("memory_per_worker_mb must be at least 1")
	}

	switch runtime.GOOS {
	case "linux":
		return loadLinuxMemoryDetails(memoryPerWorkerMB)
	default:
		return nil, fmt.Errorf("device memory detection is not supported on %s", runtime.GOOS)
	}
}

func loadLinuxMemoryDetails(memoryPerWorkerMB int) (*MemoryDetails, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("read /proc/meminfo: %w", err)
	}

	var totalKB int
	var availableKB int

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			totalKB = value
		case "MemAvailable:":
			availableKB = value
		}
	}

	if totalKB < 1 {
		return nil, fmt.Errorf("memory details: MemTotal missing from /proc/meminfo")
	}
	if availableKB < 1 {
		return nil, fmt.Errorf("memory details: MemAvailable missing from /proc/meminfo")
	}

	return &MemoryDetails{
		TotalMemoryMB:     totalKB / 1024,
		AvailableMemoryMB: availableKB / 1024,
		MemoryPerWorkerMB: memoryPerWorkerMB,
	}, nil
}

func WorkerConcurrencyFromMemory(details *MemoryDetails) (int, error) {
	if details == nil {
		return 0, fmt.Errorf("memory details are required")
	}

	memoryCap := details.AvailableMemoryMB / details.MemoryPerWorkerMB
	if memoryCap < 1 {
		memoryCap = 1
	}

	return memoryCap, nil
}
