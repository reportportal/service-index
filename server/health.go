package server

// HealthCheck specifies interface for health checks
type HealthCheck interface {
	Check() error
}

// HealthCheckFunc convenience func to deal with health checks
type HealthCheckFunc func() error

// Check checks healths
func (f HealthCheckFunc) Check() error {
	return f()
}
