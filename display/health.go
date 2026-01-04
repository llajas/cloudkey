package display

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/mem"

	"cloudkey/src/leds"
)

const (
	ThresholdWarning  = 80.0
	ThresholdCritical = 95.0
)

type HealthState int

const (
	HealthOK HealthState = iota
	HealthWarning
	HealthCritical
)

var (
	currentHealth HealthState = HealthOK
	hasUDMError   bool
	healthMonitor *leds.LEDS
)

func SetUDMError(hasError bool) {
	hasUDMError = hasError
}

func startHealthMonitor() {
	healthMonitor = &myLeds

	go func() {
		for {
			cpuPercent, _ := getCPUUsagePerCore()
			memInfo, _ := mem.VirtualMemory()
			memPercent := memInfo.UsedPercent

			newHealth := evaluateHealth(cpuPercent, memPercent)

			if newHealth != currentHealth || hasUDMError {
				currentHealth = newHealth
				updateRackLEDs(newHealth, hasUDMError)
			}

			time.Sleep(5 * time.Second)
		}
	}()

	fmt.Println("Health monitor started (CPU/RAM -> rack LED)")
}

func evaluateHealth(cpu, ram float64) HealthState {
	maxUsage := cpu
	if ram > maxUsage {
		maxUsage = ram
	}

	if maxUsage >= ThresholdCritical {
		return HealthCritical
	} else if maxUsage >= ThresholdWarning {
		return HealthWarning
	}
	return HealthOK
}

func updateRackLEDs(health HealthState, udmError bool) {
	rackBlue := myLeds.LED("rack:blue")
	rackWhite := myLeds.LED("rack:white")
	ulogo := myLeds.LED("ulogo_ctrl")

	rackBlue.Off()
	rackWhite.Off()

	if udmError || health == HealthCritical {
		rackWhite.Blink(255, 500, 500)
		fmt.Printf("Health: CRITICAL (blink white) - CPU/RAM > %.0f%% or UDM error\n", ThresholdCritical)
	} else if health == HealthWarning {
		rackWhite.On()
		fmt.Printf("Health: WARNING (solid white) - CPU/RAM > %.0f%%\n", ThresholdWarning)
	} else {
		rackBlue.On()
		fmt.Printf("Health: OK (solid blue)\n")
	}

	ulogo.On()
}
