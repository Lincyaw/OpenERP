// Package main provides the CLI entry point for the load generator.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/example/erp/tools/loadgen/internal/config"
)

// Version information (populated at build time)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// CLI flags
var (
	configPath  string
	duration    time.Duration
	concurrency int
	qps         float64
	verbose     bool
	list        bool
	validate    bool
	dryRun      bool
	showVersion bool
)

func init() {
	// Configuration
	flag.StringVar(&configPath, "config", "", "Path to the YAML configuration file (required)")
	flag.StringVar(&configPath, "c", "", "Path to the YAML configuration file (shorthand)")

	// Override flags
	flag.DurationVar(&duration, "duration", 0, "Override test duration (e.g., 5m, 1h)")
	flag.DurationVar(&duration, "d", 0, "Override test duration (shorthand)")
	flag.IntVar(&concurrency, "concurrency", 0, "Override worker pool max size")
	flag.Float64Var(&qps, "qps", 0, "Override base QPS")

	// Utility flags
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output (shorthand)")
	flag.BoolVar(&list, "list", false, "List all endpoints from configuration")
	flag.BoolVar(&list, "l", false, "List all endpoints (shorthand)")
	flag.BoolVar(&validate, "validate", false, "Validate configuration and exit")
	flag.BoolVar(&dryRun, "dry-run", false, "Parse config and show execution plan without running")
	flag.BoolVar(&showVersion, "version", false, "Show version information")

	// Custom usage
	flag.Usage = printUsage
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Load Generator - ERP System Load Testing Tool

USAGE:
    loadgen -config <path> [options]

DESCRIPTION:
    A load testing tool that generates realistic traffic patterns for the ERP system.
    It supports configurable traffic shaping, warmup phases, and comprehensive reporting.

CONFIGURATION:
    -config, -c <path>    Path to the YAML configuration file (required)

OVERRIDE OPTIONS:
    -duration, -d <dur>   Override test duration (e.g., "5m", "1h30m")
    -concurrency <n>      Override max worker pool size
    -qps <n>              Override base QPS (queries per second)

UTILITY OPTIONS:
    -list, -l             List all endpoints from configuration
    -validate             Validate configuration and exit
    -dry-run              Show execution plan without running
    -verbose, -v          Enable verbose output
    -version              Show version information
    -help, -h             Show this help message

EXAMPLES:
    # Run with default configuration
    loadgen -config configs/erp.yaml

    # Run with overridden duration and QPS
    loadgen -config configs/erp.yaml -duration 10m -qps 50

    # List all configured endpoints
    loadgen -config configs/erp.yaml -list

    # Validate configuration
    loadgen -config configs/erp.yaml -validate

    # Dry run to see execution plan
    loadgen -config configs/erp.yaml -dry-run -v

CONFIGURATION FILE FORMAT:
    The configuration file is in YAML format and supports:
    - Target system settings (baseURL, timeout, headers)
    - Authentication (bearer, API key, login flow)
    - Traffic shaping (constant, step, sine, spike, custom)
    - Rate limiting and backpressure handling
    - Warmup phase configuration
    - Endpoint definitions with weights and parameters
    - Scenarios for grouped testing

    See configs/erp.yaml for a complete example.

For more information, visit: https://github.com/example/erp/tools/loadgen
`)
}

func main() {
	flag.Parse()

	// Handle version flag
	if showVersion {
		printVersion()
		os.Exit(0)
	}

	// Validate config path is provided
	if configPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -config flag is required")
		fmt.Fprintln(os.Stderr, "")
		printUsage()
		os.Exit(1)
	}

	// Resolve config path
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving config path: %v\n", err)
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.LoadFromFile(absConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Apply CLI overrides
	applyOverrides(cfg)

	// Handle utility commands
	if validate {
		fmt.Printf("Configuration '%s' is valid.\n", cfg.Name)
		printConfigSummary(cfg)
		os.Exit(0)
	}

	if list {
		printEndpointList(cfg)
		os.Exit(0)
	}

	if dryRun {
		printExecutionPlan(cfg)
		os.Exit(0)
	}

	// Run the load test
	if err := runLoadTest(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error running load test: %v\n", err)
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Printf("loadgen version %s\n", version)
	fmt.Printf("  Build time: %s\n", buildTime)
	fmt.Printf("  Git commit: %s\n", gitCommit)
}

func applyOverrides(cfg *config.Config) {
	if duration > 0 {
		cfg.Duration = duration
		if verbose {
			fmt.Printf("Override: duration = %v\n", duration)
		}
	}

	if concurrency > 0 {
		cfg.WorkerPool.MaxSize = concurrency
		if verbose {
			fmt.Printf("Override: concurrency (workerPool.maxSize) = %d\n", concurrency)
		}
	}

	if qps > 0 {
		cfg.TrafficShaper.BaseQPS = qps
		if cfg.RateLimiter.QPS > 0 && qps < cfg.RateLimiter.QPS {
			cfg.RateLimiter.QPS = qps
		}
		if verbose {
			fmt.Printf("Override: qps = %.1f\n", qps)
		}
	}

	if verbose {
		cfg.Output.Verbose = true
	}
}

func printConfigSummary(cfg *config.Config) {
	fmt.Println()
	fmt.Println("Configuration Summary:")
	fmt.Printf("  Name:        %s\n", cfg.Name)
	fmt.Printf("  Version:     %s\n", cfg.Version)
	fmt.Printf("  Target:      %s\n", cfg.Target.BaseURL)
	fmt.Printf("  Duration:    %v\n", cfg.Duration)
	fmt.Printf("  Auth Type:   %s\n", getAuthType(cfg))
	fmt.Printf("  Endpoints:   %d\n", len(cfg.Endpoints))
	fmt.Printf("  Scenarios:   %d\n", len(cfg.Scenarios))
	fmt.Printf("  TrafficType: %s\n", cfg.TrafficShaper.Type)
	fmt.Printf("  Base QPS:    %.1f\n", cfg.TrafficShaper.BaseQPS)
}

func getAuthType(cfg *config.Config) string {
	if cfg.Auth.Type == "" {
		return "none"
	}
	return cfg.Auth.Type
}

func printEndpointList(cfg *config.Config) {
	fmt.Printf("Endpoints in '%s' (%d total):\n", cfg.Name, len(cfg.Endpoints))
	fmt.Println()

	// Group by tags for better readability
	tagGroups := make(map[string][]config.EndpointConfig)
	for _, ep := range cfg.Endpoints {
		category := "other"
		if len(ep.Tags) > 0 {
			category = ep.Tags[0]
		}
		tagGroups[category] = append(tagGroups[category], ep)
	}

	// Sort categories for deterministic output
	categories := make([]string, 0, len(tagGroups))
	for category := range tagGroups {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	// Print grouped endpoints
	for _, category := range categories {
		endpoints := tagGroups[category]
		fmt.Printf("== %s ==\n", strings.ToUpper(category))
		for _, ep := range endpoints {
			status := ""
			if ep.Disabled {
				status = " [DISABLED]"
			}
			authRequired := "AUTH"
			if ep.RequiresAuth != nil && !*ep.RequiresAuth {
				authRequired = "NOAUTH"
			}
			fmt.Printf("  %-40s %s %-6s w:%-3d %s%s\n",
				ep.Name,
				ep.Method,
				ep.Path,
				ep.Weight,
				authRequired,
				status,
			)
		}
		fmt.Println()
	}

	// Print summary
	fmt.Println("Summary:")
	fmt.Printf("  Total Weight: %d\n", cfg.TotalWeight())
	enabledCount := 0
	readCount := 0
	writeCount := 0
	for _, ep := range cfg.Endpoints {
		if !ep.Disabled {
			enabledCount++
			switch ep.Method {
			case "GET":
				readCount++
			case "POST", "PUT", "PATCH", "DELETE":
				writeCount++
			}
		}
	}
	fmt.Printf("  Enabled:      %d\n", enabledCount)
	fmt.Printf("  Read (GET):   %d\n", readCount)
	fmt.Printf("  Write:        %d\n", writeCount)
}

func printExecutionPlan(cfg *config.Config) {
	fmt.Println("=== Execution Plan (Dry Run) ===")
	fmt.Println()

	// Configuration summary
	printConfigSummary(cfg)

	// Traffic shaping plan
	fmt.Println()
	fmt.Println("Traffic Shaping:")
	fmt.Printf("  Type:     %s\n", cfg.TrafficShaper.Type)
	fmt.Printf("  Base QPS: %.1f\n", cfg.TrafficShaper.BaseQPS)

	switch cfg.TrafficShaper.Type {
	case "step":
		if cfg.TrafficShaper.Step != nil {
			fmt.Println("  Steps:")
			totalTime := time.Duration(0)
			for i, step := range cfg.TrafficShaper.Step.Steps {
				fmt.Printf("    %d. QPS: %.1f, Duration: %v, Ramp: %v\n",
					i+1, step.QPS, step.Duration, step.RampDuration)
				totalTime += step.Duration + step.RampDuration
			}
			fmt.Printf("  Total step time: %v\n", totalTime)
		}
	case "sine":
		fmt.Printf("  Amplitude: %.1f\n", cfg.TrafficShaper.Amplitude)
		fmt.Printf("  Period:    %v\n", cfg.TrafficShaper.Period)
	case "spike":
		if cfg.TrafficShaper.Spike != nil {
			fmt.Printf("  Spike QPS: %.1f\n", cfg.TrafficShaper.Spike.SpikeQPS)
			fmt.Printf("  Duration:  %v\n", cfg.TrafficShaper.Spike.SpikeDuration)
			fmt.Printf("  Interval:  %v\n", cfg.TrafficShaper.Spike.SpikeInterval)
		}
	}

	// Worker pool
	fmt.Println()
	fmt.Println("Worker Pool:")
	fmt.Printf("  Min:     %d\n", cfg.WorkerPool.MinSize)
	fmt.Printf("  Max:     %d\n", cfg.WorkerPool.MaxSize)
	fmt.Printf("  Initial: %d\n", cfg.WorkerPool.InitialSize)

	// Warmup phase
	fmt.Println()
	fmt.Println("Warmup Phase:")
	fmt.Printf("  Enabled:    %v\n", cfg.Warmup.Iterations > 0)
	if cfg.Warmup.Iterations > 0 {
		fmt.Printf("  Iterations: %d\n", cfg.Warmup.Iterations)
		fmt.Printf("  Timeout:    %v\n", cfg.Warmup.Timeout)
		fmt.Printf("  Fill types: %d\n", len(cfg.Warmup.Fill))
		if verbose {
			for _, fill := range cfg.Warmup.Fill {
				fmt.Printf("    - %s\n", fill)
			}
		}
	}

	// Endpoint distribution
	fmt.Println()
	fmt.Println("Endpoint Distribution (top 10 by weight):")
	type epWeight struct {
		name   string
		weight int
	}
	weights := make([]epWeight, 0, len(cfg.Endpoints))
	for _, ep := range cfg.Endpoints {
		if !ep.Disabled {
			weights = append(weights, epWeight{ep.Name, ep.Weight})
		}
	}
	// Sort by weight descending using sort.Slice
	sort.Slice(weights, func(i, j int) bool {
		return weights[i].weight > weights[j].weight
	})
	totalWeight := cfg.TotalWeight()
	shown := min(10, len(weights))
	for i := range shown {
		pct := float64(weights[i].weight) / float64(totalWeight) * 100
		fmt.Printf("  %-40s w:%-3d (%.1f%%)\n", weights[i].name, weights[i].weight, pct)
	}
	if len(weights) > shown {
		fmt.Printf("  ... and %d more endpoints\n", len(weights)-shown)
	}

	fmt.Println()
	fmt.Println("Ready to execute. Remove -dry-run flag to start the load test.")
}

func runLoadTest(cfg *config.Config) error {
	// For now, print a placeholder message
	// The actual load test runner will be implemented in LOADGEN-003
	fmt.Println("=== Load Test ===")
	fmt.Printf("Configuration: %s\n", cfg.Name)
	fmt.Printf("Target: %s\n", cfg.Target.BaseURL)
	fmt.Printf("Duration: %v\n", cfg.Duration)
	fmt.Printf("Endpoints: %d\n", len(cfg.Endpoints))
	fmt.Println()
	fmt.Println("Load test runner not yet implemented.")
	fmt.Println("This CLI framework supports the following features:")
	fmt.Println("  - Configuration loading and validation")
	fmt.Println("  - Command-line overrides (-duration, -concurrency, -qps)")
	fmt.Println("  - Endpoint listing (-list)")
	fmt.Println("  - Dry run mode (-dry-run)")
	fmt.Println()
	fmt.Println("The actual load test execution will be implemented in subsequent tasks.")
	return nil
}
