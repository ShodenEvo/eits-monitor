package collector

type cpuTimes struct{ idle, total uint64 }

func calculateCPU(previous, current cpuTimes) float64 {
	if previous.total == 0 || current.total <= previous.total {
		return 0
	}
	totalDelta := current.total - previous.total
	idleDelta := current.idle - previous.idle
	if idleDelta > totalDelta {
		return 0
	}
	return 100 * float64(totalDelta-idleDelta) / float64(totalDelta)
}
