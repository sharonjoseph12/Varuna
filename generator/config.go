package generator

// DefaultConfig returns a config suitable for the live demo and benchmark.
// VesselCount=200 comfortably sustains ≥50k msg/sec with coastal cadence.
func DefaultConfig() Config {
	return Config{
		VesselCount:  200,
		OutputBuffer: 10000,
		TriggerPort:  8081,
		DataDir:      "data",
	}
}
