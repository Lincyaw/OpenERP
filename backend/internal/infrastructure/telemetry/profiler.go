// Package telemetry provides Pyroscope continuous profiling integration.
package telemetry

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/grafana/pyroscope-go"
	"go.uber.org/zap"
)

// ProfilerConfig holds Pyroscope continuous profiling configuration.
type ProfilerConfig struct {
	Enabled           bool   // Enable continuous profiling
	ServerAddress     string // Pyroscope server address (e.g., "http://pyroscope:4040")
	ApplicationName   string // Application name for profiles
	BasicAuthUser     string // Optional: Basic auth username (for Grafana Cloud)
	BasicAuthPassword string // Optional: Basic auth password (for Grafana Cloud)

	// Profile types to enable
	ProfileCPU           bool // CPU profiling
	ProfileAllocObjects  bool // Allocation objects profiling
	ProfileAllocSpace    bool // Allocation space profiling
	ProfileInuseObjects  bool // In-use objects profiling
	ProfileInuseSpace    bool // In-use space profiling
	ProfileGoroutines    bool // Goroutine profiling
	ProfileMutexCount    bool // Mutex count profiling
	ProfileMutexDuration bool // Mutex duration profiling
	ProfileBlockCount    bool // Block count profiling
	ProfileBlockDuration bool // Block duration profiling

	// Advanced options
	MutexProfileFraction int  // Mutex profile fraction (default: 5)
	BlockProfileRate     int  // Block profile rate (default: 5)
	DisableGCRuns        bool // Disable GC runs in profiler
}

// Profiler wraps the Pyroscope profiler with lifecycle management.
type Profiler struct {
	profiler *pyroscope.Profiler
	logger   *zap.Logger
	config   ProfilerConfig
	mu       sync.Mutex
	stopped  bool
}

// NewProfiler creates and starts a new Pyroscope profiler.
// If profiling is disabled, it returns a no-op profiler.
func NewProfiler(cfg ProfilerConfig, logger *zap.Logger) (*Profiler, error) {
	p := &Profiler{
		logger: logger,
		config: cfg,
	}

	if !cfg.Enabled {
		logger.Info("Continuous profiling disabled, using no-op profiler")
		return p, nil
	}

	// Validate required fields
	if cfg.ServerAddress == "" {
		return nil, fmt.Errorf("profiler server address is required when profiling is enabled")
	}
	if cfg.ApplicationName == "" {
		return nil, fmt.Errorf("profiler application name is required when profiling is enabled")
	}

	// Configure runtime settings for mutex and block profiling
	if cfg.ProfileMutexCount || cfg.ProfileMutexDuration {
		fraction := cfg.MutexProfileFraction
		if fraction <= 0 {
			fraction = 5
		}
		runtime.SetMutexProfileFraction(fraction)
		logger.Debug("Mutex profiling enabled", zap.Int("fraction", fraction))
	}

	if cfg.ProfileBlockCount || cfg.ProfileBlockDuration {
		rate := cfg.BlockProfileRate
		if rate <= 0 {
			rate = 5
		}
		runtime.SetBlockProfileRate(rate)
		logger.Debug("Block profiling enabled", zap.Int("rate", rate))
	}

	// Build profile types list
	profileTypes := p.buildProfileTypes()
	if len(profileTypes) == 0 {
		logger.Warn("No profile types enabled, profiler will not collect any data")
	}

	// Build tags (labels) for profiles
	tags := map[string]string{}
	if hostname := os.Getenv("HOSTNAME"); hostname != "" {
		tags["hostname"] = hostname
	}
	if podName := os.Getenv("POD_NAME"); podName != "" {
		tags["pod"] = podName
	}

	// Create Pyroscope configuration
	pyroscopeCfg := pyroscope.Config{
		ApplicationName: cfg.ApplicationName,
		ServerAddress:   cfg.ServerAddress,
		Logger:          newPyroscopeLogger(logger),
		Tags:            tags,
		ProfileTypes:    profileTypes,
		DisableGCRuns:   cfg.DisableGCRuns,
	}

	// Add basic auth if configured (for Grafana Cloud)
	if cfg.BasicAuthUser != "" && cfg.BasicAuthPassword != "" {
		pyroscopeCfg.BasicAuthUser = cfg.BasicAuthUser
		pyroscopeCfg.BasicAuthPassword = cfg.BasicAuthPassword
	}

	// Start the profiler
	profiler, err := pyroscope.Start(pyroscopeCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to start Pyroscope profiler: %w", err)
	}

	p.profiler = profiler

	logger.Info("Pyroscope profiler started",
		zap.String("server_address", cfg.ServerAddress),
		zap.String("application_name", cfg.ApplicationName),
		zap.Int("profile_types", len(profileTypes)),
		zap.Bool("disable_gc_runs", cfg.DisableGCRuns),
	)

	return p, nil
}

// buildProfileTypes constructs the list of enabled profile types.
func (p *Profiler) buildProfileTypes() []pyroscope.ProfileType {
	var types []pyroscope.ProfileType

	if p.config.ProfileCPU {
		types = append(types, pyroscope.ProfileCPU)
	}
	if p.config.ProfileAllocObjects {
		types = append(types, pyroscope.ProfileAllocObjects)
	}
	if p.config.ProfileAllocSpace {
		types = append(types, pyroscope.ProfileAllocSpace)
	}
	if p.config.ProfileInuseObjects {
		types = append(types, pyroscope.ProfileInuseObjects)
	}
	if p.config.ProfileInuseSpace {
		types = append(types, pyroscope.ProfileInuseSpace)
	}
	if p.config.ProfileGoroutines {
		types = append(types, pyroscope.ProfileGoroutines)
	}
	if p.config.ProfileMutexCount {
		types = append(types, pyroscope.ProfileMutexCount)
	}
	if p.config.ProfileMutexDuration {
		types = append(types, pyroscope.ProfileMutexDuration)
	}
	if p.config.ProfileBlockCount {
		types = append(types, pyroscope.ProfileBlockCount)
	}
	if p.config.ProfileBlockDuration {
		types = append(types, pyroscope.ProfileBlockDuration)
	}

	return types
}

// Stop gracefully stops the profiler, flushing any pending profiles.
// It is safe to call Stop multiple times.
//
// Note: Unlike TracerProvider.Shutdown() and MeterProvider.Shutdown() which accept
// a context with timeout, the Pyroscope SDK's Stop() method does not support
// context-based cancellation. This means Stop() could potentially block if the
// Pyroscope server is unresponsive. In practice, the SDK implements internal
// timeouts to prevent indefinite blocking.
func (p *Profiler) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		p.logger.Debug("Profiler already stopped")
		return nil
	}

	p.stopped = true

	if p.profiler == nil {
		p.logger.Debug("No profiler to stop (profiling disabled)")
		return nil
	}

	p.logger.Info("Stopping Pyroscope profiler...")

	if err := p.profiler.Stop(); err != nil {
		p.logger.Error("Error stopping profiler", zap.Error(err))
		return fmt.Errorf("failed to stop profiler: %w", err)
	}

	p.logger.Info("Pyroscope profiler stopped successfully")
	return nil
}

// IsEnabled returns whether profiling is enabled.
func (p *Profiler) IsEnabled() bool {
	return p.config.Enabled && p.profiler != nil
}

// GetConfig returns a copy of the profiler configuration.
func (p *Profiler) GetConfig() ProfilerConfig {
	return p.config
}

// pyroscopeLogger adapts zap.Logger to pyroscope.Logger interface.
type pyroscopeLogger struct {
	logger *zap.Logger
}

// newPyroscopeLogger creates a new pyroscope logger adapter.
func newPyroscopeLogger(logger *zap.Logger) pyroscope.Logger {
	return &pyroscopeLogger{logger: logger.Named("pyroscope")}
}

// Infof logs an info message.
func (l *pyroscopeLogger) Infof(format string, args ...any) {
	l.logger.Sugar().Infof(format, args...)
}

// Debugf logs a debug message.
func (l *pyroscopeLogger) Debugf(format string, args ...any) {
	l.logger.Sugar().Debugf(format, args...)
}

// Errorf logs an error message.
func (l *pyroscopeLogger) Errorf(format string, args ...any) {
	l.logger.Sugar().Errorf(format, args...)
}
