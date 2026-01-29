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

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/config"
	"github.com/example/erp/tools/loadgen/internal/parser"
)

// Version information (populated at build time)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// CLI flags
var (
	configPath     string
	duration       time.Duration
	concurrency    int
	qps            float64
	verbose        bool
	list           bool
	validate       bool
	dryRun         bool
	showVersion    bool
	openapiPath    string
	inferDryRun    bool
	minConfidence  float64
	outputFormat   string
	outputFile     string
	prometheusAddr string
)

func init() {
	// Configuration
	flag.StringVar(&configPath, "config", "", "Path to the YAML configuration file")
	flag.StringVar(&configPath, "c", "", "Path to the YAML configuration file (shorthand)")

	// OpenAPI parsing
	flag.StringVar(&openapiPath, "openapi", "", "Path to OpenAPI/Swagger spec file (for endpoint discovery)")
	flag.StringVar(&openapiPath, "o", "", "Path to OpenAPI/Swagger spec file (shorthand)")

	// Override flags
	flag.DurationVar(&duration, "duration", 0, "Override test duration (e.g., 5m, 1h)")
	flag.DurationVar(&duration, "d", 0, "Override test duration (shorthand)")
	flag.IntVar(&concurrency, "concurrency", 0, "Override worker pool max size")
	flag.Float64Var(&qps, "qps", 0, "Override base QPS")

	// Utility flags
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output (shorthand)")
	flag.BoolVar(&list, "list", false, "List all endpoints from configuration or OpenAPI spec")
	flag.BoolVar(&list, "l", false, "List all endpoints (shorthand)")
	flag.BoolVar(&validate, "validate", false, "Validate configuration and exit")
	flag.BoolVar(&dryRun, "dry-run", false, "Parse config and show execution plan without running")
	flag.BoolVar(&showVersion, "version", false, "Show version information")

	// Inference flags
	flag.BoolVar(&inferDryRun, "infer", false, "Run semantic type inference on OpenAPI spec (dry-run mode)")
	flag.Float64Var(&minConfidence, "min-confidence", 0.7, "Minimum confidence threshold for inference (0.0-1.0)")

	// Output flags
	flag.StringVar(&outputFormat, "output", "", "Output format: console, json, or console,json (enables JSON report)")
	flag.StringVar(&outputFile, "output-file", "", "JSON output file path (overrides config, supports {{.Timestamp}})")
	flag.StringVar(&prometheusAddr, "prometheus", "", "Prometheus metrics endpoint (e.g., :9090 or localhost:9090)")

	// Custom usage
	flag.Usage = printUsage
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Load Generator - ERP System Load Testing Tool

USAGE:
    loadgen -config <path> [options]
    loadgen -openapi <path> -list      (Parse and list OpenAPI endpoints)

DESCRIPTION:
    A load testing tool that generates realistic traffic patterns for the ERP system.
    It supports configurable traffic shaping, warmup phases, and comprehensive reporting.

    The tool can parse OpenAPI/Swagger specifications to discover endpoints and their
    parameters, which can then be used to generate realistic test traffic.

CONFIGURATION:
    -config, -c <path>    Path to the YAML configuration file
    -openapi, -o <path>   Path to OpenAPI/Swagger spec file (YAML or JSON)

OVERRIDE OPTIONS:
    -duration, -d <dur>   Override test duration (e.g., "5m", "1h30m")
    -concurrency <n>      Override max worker pool size
    -qps <n>              Override base QPS (queries per second)

UTILITY OPTIONS:
    -list, -l             List all endpoints from configuration or OpenAPI spec
    -validate             Validate configuration and exit
    -dry-run              Show execution plan without running
    -verbose, -v          Enable verbose output
    -version              Show version information
    -help, -h             Show this help message

OUTPUT OPTIONS:
    -output <format>      Output format: console, json, or console,json
    -output-file <path>   JSON output file (supports {{.Timestamp}} template)
    -prometheus <addr>    Enable Prometheus metrics endpoint (e.g., :9090)

EXAMPLES:
    # Run with default configuration
    loadgen -config configs/erp.yaml

    # Run with overridden duration and QPS
    loadgen -config configs/erp.yaml -duration 10m -qps 50

    # Generate JSON report
    loadgen -config configs/erp.yaml -output json

    # Generate JSON report with custom file path
    loadgen -config configs/erp.yaml -output json -output-file results/test-{{.Timestamp}}.json

    # Enable Prometheus metrics endpoint
    loadgen -config configs/erp.yaml -prometheus :9090

    # List all configured endpoints
    loadgen -config configs/erp.yaml -list

    # Parse and list OpenAPI endpoints (for endpoint discovery)
    loadgen -openapi backend/docs/swagger.yaml -list

    # Parse OpenAPI with verbose output (show parameters and response fields)
    loadgen -openapi backend/docs/swagger.yaml -list -v

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

OPENAPI PARSING:
    The OpenAPI parser supports both Swagger 2.0 and OpenAPI 3.0 specifications.
    It extracts:
    - All HTTP endpoints with their methods and paths
    - Input parameters (path, query, header, body)
    - Response schema output fields
    - Security requirements
    - Tags for categorization

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

	// Handle OpenAPI parsing mode
	if openapiPath != "" {
		handleOpenAPIMode()
		return
	}

	// Config mode - validate config path is provided
	if configPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -config or -openapi flag is required")
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

	// Apply output format override
	if outputFormat != "" {
		cfg.Output.Type = outputFormat
		// Enable JSON output if format includes "json"
		if strings.Contains(strings.ToLower(outputFormat), "json") {
			cfg.Output.JSON.Enabled = true
			if verbose {
				fmt.Printf("Override: output format = %s (JSON enabled)\n", outputFormat)
			}
		}
	}

	// Apply output file override
	if outputFile != "" {
		cfg.Output.JSON.Enabled = true
		cfg.Output.JSON.File = outputFile
		if verbose {
			fmt.Printf("Override: output file = %s\n", outputFile)
		}
	}

	// Apply Prometheus override
	if prometheusAddr != "" {
		cfg.Output.Prometheus.Enabled = true
		// Parse address - support both :9090 and localhost:9090 formats
		port := parsePrometheusPort(prometheusAddr)
		if port > 0 {
			cfg.Output.Prometheus.Port = port
		}
		if cfg.Output.Prometheus.Path == "" {
			cfg.Output.Prometheus.Path = "/metrics"
		}
		if verbose {
			fmt.Printf("Override: Prometheus enabled on port %d\n", cfg.Output.Prometheus.Port)
		}
	}
}

// parsePrometheusPort extracts port from address string.
// Supports formats: :9090, localhost:9090, 9090
// Returns 0 for invalid ports (including out of range 1-65535).
func parsePrometheusPort(addr string) int {
	addr = strings.TrimSpace(addr)

	// Handle just port number
	if !strings.Contains(addr, ":") {
		var port int
		if _, err := fmt.Sscanf(addr, "%d", &port); err == nil {
			if port > 0 && port <= 65535 {
				return port
			}
		}
		return 0
	}

	// Handle :port or host:port
	parts := strings.Split(addr, ":")
	if len(parts) >= 2 {
		var port int
		if _, err := fmt.Sscanf(parts[len(parts)-1], "%d", &port); err == nil {
			if port > 0 && port <= 65535 {
				return port
			}
		}
	}
	return 0
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

// handleOpenAPIMode handles OpenAPI parsing mode
func handleOpenAPIMode() {
	absPath, err := filepath.Abs(openapiPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving OpenAPI path: %v\n", err)
		os.Exit(1)
	}

	p := parser.NewParser()
	spec, err := p.ParseFile(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	// Handle inference dry-run mode
	if inferDryRun {
		printInferenceResults(spec)
		os.Exit(0)
	}

	if list {
		printOpenAPIEndpointList(spec)
		os.Exit(0)
	}

	// Default: print summary
	fmt.Print(spec.Summary())
	os.Exit(0)
}

// printOpenAPIEndpointList prints endpoints from an OpenAPI spec
func printOpenAPIEndpointList(spec *parser.OpenAPISpec) {
	fmt.Printf("OpenAPI Endpoints from '%s' (v%s)\n", spec.Title, spec.Version)
	if spec.BasePath != "" {
		fmt.Printf("Base Path: %s\n", spec.BasePath)
	}
	fmt.Printf("Total: %d endpoints\n", len(spec.Endpoints))
	fmt.Println()

	// Group by tags for better readability
	tagGroups := make(map[string][]parser.EndpointUnit)
	for _, ep := range spec.Endpoints {
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
		tagDesc := ""
		if desc, ok := spec.Tags[category]; ok && desc != "" {
			tagDesc = fmt.Sprintf(" - %s", desc)
		}
		fmt.Printf("== %s%s (%d) ==\n", strings.ToUpper(category), tagDesc, len(endpoints))

		for _, ep := range endpoints {
			status := ""
			if ep.Deprecated {
				status = " [DEPRECATED]"
			}
			authRequired := "AUTH"
			if !ep.RequiresAuth {
				authRequired = "NOAUTH"
			}

			// Print basic info
			fmt.Printf("  %-7s %-50s %s%s\n",
				ep.Method,
				ep.Path,
				authRequired,
				status,
			)

			// Print verbose details
			if verbose {
				if ep.Summary != "" {
					fmt.Printf("          Summary: %s\n", ep.Summary)
				}
				if ep.OperationID != "" {
					fmt.Printf("          OperationID: %s\n", ep.OperationID)
				}

				// Print input pins
				if len(ep.InputPins) > 0 {
					fmt.Printf("          Input Parameters:\n")
					for _, pin := range ep.InputPins {
						required := ""
						if pin.Required {
							required = "*"
						}
						fmt.Printf("            - %s%s (%s, %s)\n",
							pin.Name, required, pin.Location, pin.Type)
					}
				}

				// Print output pins (limited to first 5)
				if len(ep.OutputPins) > 0 {
					fmt.Printf("          Output Fields:\n")
					shown := min(5, len(ep.OutputPins))
					for i := range shown {
						pin := ep.OutputPins[i]
						fmt.Printf("            - %s: %s (%s)\n",
							pin.JSONPath, pin.Type, pin.Name)
					}
					if len(ep.OutputPins) > shown {
						fmt.Printf("            ... and %d more\n", len(ep.OutputPins)-shown)
					}
				}

				fmt.Println()
			}
		}
		fmt.Println()
	}

	// Print summary
	fmt.Println("Summary:")
	authCount := 0
	methodCounts := make(map[string]int)
	for _, ep := range spec.Endpoints {
		if ep.RequiresAuth {
			authCount++
		}
		methodCounts[ep.Method]++
	}
	fmt.Printf("  Authenticated: %d\n", authCount)
	fmt.Printf("  Public:        %d\n", len(spec.Endpoints)-authCount)

	fmt.Println("\n  By Method:")
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		if count, ok := methodCounts[method]; ok {
			fmt.Printf("    %s: %d\n", method, count)
		}
	}

	// Security definitions
	if len(spec.SecurityDefinitions) > 0 {
		fmt.Println("\n  Security Schemes:")
		for name, scheme := range spec.SecurityDefinitions {
			fmt.Printf("    %s: %s", name, scheme.Type)
			if scheme.In != "" {
				fmt.Printf(" (in %s)", scheme.In)
			}
			fmt.Println()
		}
	}
}

// printInferenceResults runs semantic type inference and displays results
func printInferenceResults(spec *parser.OpenAPISpec) {
	fmt.Printf("=== Semantic Type Inference Results ===\n")
	fmt.Printf("OpenAPI: '%s' (v%s)\n", spec.Title, spec.Version)
	fmt.Printf("Minimum Confidence: %.0f%%\n", minConfidence*100)
	fmt.Println()

	// Create inference engine
	engine := parser.NewSemanticInferenceEngine()
	engine.SetMinConfidence(minConfidence)

	// Run inference on all endpoints
	registry := engine.InferSpec(spec)

	// Calculate and display statistics
	stats := parser.CalculateStats(registry)

	fmt.Printf("=== Summary ===\n")
	fmt.Printf("Total Fields:        %d\n", stats.TotalFields)
	fmt.Printf("Inferred Fields:     %d (%.1f%%)\n", stats.InferredFields,
		float64(stats.InferredFields)/float64(stats.TotalFields)*100)
	fmt.Printf("Unknown Fields:      %d (%.1f%%)\n", stats.UnknownFields,
		float64(stats.UnknownFields)/float64(stats.TotalFields)*100)
	fmt.Printf("High Confidence:     %d (>=90%%)\n", stats.HighConfidence)
	fmt.Printf("Medium Confidence:   %d (70-89%%)\n", stats.MediumConfidence)
	fmt.Printf("Low Confidence:      %d (<70%%)\n", stats.LowConfidence)
	fmt.Printf("Estimated Accuracy:  %.1f%%\n", stats.AccuracyEstimate)
	fmt.Println()

	// Display by source
	fmt.Printf("=== By Inference Source ===\n")
	for source, count := range stats.BySource {
		fmt.Printf("  %-20s %d\n", source+":", count)
	}
	fmt.Println()

	// Display by category
	fmt.Printf("=== By Semantic Category ===\n")
	for category, count := range stats.ByCategory {
		fmt.Printf("  %-20s %d\n", category+":", count)
	}
	fmt.Println()

	// Verbose mode: show all inferences
	if verbose {
		fmt.Printf("=== Detailed Inference Results ===\n\n")

		// Group by endpoint
		endpointPins := make(map[string][]*circuit.Pin)
		for _, pin := range registry.AllPins {
			key := pin.Location.EndpointMethod + " " + pin.Location.EndpointPath
			endpointPins[key] = append(endpointPins[key], pin)
		}

		// Sort endpoints
		endpoints := make([]string, 0, len(endpointPins))
		for ep := range endpointPins {
			endpoints = append(endpoints, ep)
		}
		sort.Strings(endpoints)

		for _, ep := range endpoints {
			pins := endpointPins[ep]
			fmt.Printf("--- %s ---\n", ep)

			// Separate inputs and outputs
			var inputs, outputs []*circuit.Pin
			for _, pin := range pins {
				if pin.IsInput() {
					inputs = append(inputs, pin)
				} else {
					outputs = append(outputs, pin)
				}
			}

			if len(inputs) > 0 {
				fmt.Printf("  Inputs:\n")
				for _, pin := range inputs {
					confidence := fmt.Sprintf("%.0f%%", pin.InferenceConfidence*100)
					fmt.Printf("    %-25s -> %-30s [%s, %s]\n",
						pin.Name, pin.SemanticType, confidence, pin.InferenceSource)
				}
			}

			if len(outputs) > 0 {
				fmt.Printf("  Outputs:\n")
				for _, pin := range outputs {
					confidence := fmt.Sprintf("%.0f%%", pin.InferenceConfidence*100)
					fmt.Printf("    %-25s -> %-30s [%s, %s]\n",
						pin.Name, pin.SemanticType, confidence, pin.InferenceSource)
				}
			}
			fmt.Println()
		}
	}

	// Show connections
	connections := registry.GetConnections()
	fmt.Printf("=== Producer-Consumer Connections ===\n")
	fmt.Printf("Total Connections: %d\n", len(connections))

	if verbose && len(connections) > 0 {
		// Group by semantic type
		connByType := make(map[circuit.SemanticType][]circuit.PinConnection)
		for _, conn := range connections {
			connByType[conn.Producer.SemanticType] = append(connByType[conn.Producer.SemanticType], conn)
		}

		fmt.Println()
		for semType, conns := range connByType {
			fmt.Printf("  %s (%d connections):\n", semType, len(conns))
			shown := min(3, len(conns))
			for i := 0; i < shown; i++ {
				conn := conns[i]
				fmt.Printf("    %s %s -> %s %s\n",
					conn.Producer.Location.EndpointMethod,
					conn.Producer.Location.EndpointPath,
					conn.Consumer.Location.EndpointMethod,
					conn.Consumer.Location.EndpointPath)
			}
			if len(conns) > shown {
				fmt.Printf("    ... and %d more\n", len(conns)-shown)
			}
		}
	}

	// Show unconnected pins
	unconnectedInputs := registry.GetUnconnectedInputs()
	unconnectedOutputs := registry.GetUnconnectedOutputs()

	fmt.Println()
	fmt.Printf("=== Unconnected Pins ===\n")
	fmt.Printf("Unconnected Inputs:  %d (need producers)\n", len(unconnectedInputs))
	fmt.Printf("Unconnected Outputs: %d (no consumers)\n", len(unconnectedOutputs))

	if verbose && len(unconnectedInputs) > 0 {
		fmt.Println("\nUnconnected Input Types (need data generators):")
		typeCount := make(map[circuit.SemanticType]int)
		for _, pin := range unconnectedInputs {
			typeCount[pin.SemanticType]++
		}
		for semType, count := range typeCount {
			fmt.Printf("  %-30s %d\n", semType, count)
		}
	}
}
