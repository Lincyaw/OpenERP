// Package circuit provides circuit-board-like components for the load generator.
// This file defines Pin and PinLocation types for connecting API inputs and outputs.
package circuit

// PinDirection indicates whether a pin is an input or output.
type PinDirection string

const (
	// PinDirectionInput represents an input pin (parameter).
	PinDirectionInput PinDirection = "input"
	// PinDirectionOutput represents an output pin (response field).
	PinDirectionOutput PinDirection = "output"
)

// PinLocation describes where a pin is located in an API endpoint.
type PinLocation struct {
	// EndpointPath is the API endpoint path (e.g., "/customers/{id}").
	EndpointPath string `json:"endpointPath" yaml:"endpointPath"`

	// EndpointMethod is the HTTP method (GET, POST, PUT, DELETE, PATCH).
	EndpointMethod string `json:"endpointMethod" yaml:"endpointMethod"`

	// OperationID is the OpenAPI operation ID if available.
	OperationID string `json:"operationId,omitempty" yaml:"operationId,omitempty"`

	// Direction indicates if this is an input or output pin.
	Direction PinDirection `json:"direction" yaml:"direction"`

	// ParameterLocation is where the parameter is found (path, query, header, body).
	// Only applicable for input pins.
	ParameterLocation string `json:"parameterLocation,omitempty" yaml:"parameterLocation,omitempty"`

	// JSONPath is the path to extract/set the value.
	// For inputs: path in request body (e.g., "$.customer_id")
	// For outputs: path in response body (e.g., "$.data.id")
	JSONPath string `json:"jsonPath,omitempty" yaml:"jsonPath,omitempty"`

	// FieldName is the parameter or field name.
	FieldName string `json:"fieldName" yaml:"fieldName"`
}

// Pin represents a connection point for data flow between API endpoints.
// Pins with matching SemanticTypes can be connected to form producer-consumer relationships.
type Pin struct {
	// ID is a unique identifier for this pin.
	ID string `json:"id" yaml:"id"`

	// Name is a human-readable name for the pin.
	Name string `json:"name" yaml:"name"`

	// SemanticType classifies what kind of data this pin handles.
	SemanticType SemanticType `json:"semanticType" yaml:"semanticType"`

	// Location describes where this pin is in the API.
	Location PinLocation `json:"location" yaml:"location"`

	// DataType is the underlying data type (string, integer, number, boolean, array, object).
	DataType string `json:"dataType" yaml:"dataType"`

	// Format is the format hint (uuid, date-time, email, etc.).
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// Required indicates if this pin must have a value (for input pins).
	Required bool `json:"required,omitempty" yaml:"required,omitempty"`

	// Description provides context about the pin.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// InferenceConfidence is the confidence level of semantic type inference (0.0-1.0).
	// 1.0 means explicitly configured, lower values indicate inferred types.
	InferenceConfidence float64 `json:"inferenceConfidence,omitempty" yaml:"inferenceConfidence,omitempty"`

	// InferenceSource describes how the semantic type was determined.
	// Values: "explicit", "field_name", "endpoint_path", "format", "default"
	InferenceSource string `json:"inferenceSource,omitempty" yaml:"inferenceSource,omitempty"`
}

// NewInputPin creates a new input pin with the given parameters.
func NewInputPin(endpointPath, method, fieldName string, semanticType SemanticType) *Pin {
	return &Pin{
		ID:           generatePinID(endpointPath, method, "input", fieldName),
		Name:         fieldName,
		SemanticType: semanticType,
		Location: PinLocation{
			EndpointPath:   endpointPath,
			EndpointMethod: method,
			Direction:      PinDirectionInput,
			FieldName:      fieldName,
		},
	}
}

// NewOutputPin creates a new output pin with the given parameters.
func NewOutputPin(endpointPath, method, fieldName, jsonPath string, semanticType SemanticType) *Pin {
	return &Pin{
		ID:           generatePinID(endpointPath, method, "output", fieldName),
		Name:         fieldName,
		SemanticType: semanticType,
		Location: PinLocation{
			EndpointPath:   endpointPath,
			EndpointMethod: method,
			Direction:      PinDirectionOutput,
			FieldName:      fieldName,
			JSONPath:       jsonPath,
		},
	}
}

// generatePinID creates a unique ID for a pin.
func generatePinID(path, method, direction, field string) string {
	return method + ":" + path + ":" + direction + ":" + field
}

// IsInput returns true if this is an input pin.
func (p *Pin) IsInput() bool {
	return p.Location.Direction == PinDirectionInput
}

// IsOutput returns true if this is an output pin.
func (p *Pin) IsOutput() bool {
	return p.Location.Direction == PinDirectionOutput
}

// IsHighConfidence returns true if the semantic type inference has high confidence (>= 0.8).
func (p *Pin) IsHighConfidence() bool {
	return p.InferenceConfidence >= 0.8
}

// CanConnectTo returns true if this output pin can connect to the given input pin.
// Pins can connect if they have the same semantic type.
func (p *Pin) CanConnectTo(other *Pin) bool {
	if p == nil || other == nil {
		return false
	}
	// Output pins connect to input pins
	if !p.IsOutput() || !other.IsInput() {
		return false
	}
	// Must have matching semantic types
	return p.SemanticType == other.SemanticType && p.SemanticType != "" && p.SemanticType != UnknownSemanticType
}

// PinRegistry holds all pins discovered from an API specification.
type PinRegistry struct {
	// InputPins maps semantic type to input pins that consume that type.
	InputPins map[SemanticType][]*Pin

	// OutputPins maps semantic type to output pins that produce that type.
	OutputPins map[SemanticType][]*Pin

	// AllPins is a flat list of all pins.
	AllPins []*Pin
}

// NewPinRegistry creates a new empty pin registry.
func NewPinRegistry() *PinRegistry {
	return &PinRegistry{
		InputPins:  make(map[SemanticType][]*Pin),
		OutputPins: make(map[SemanticType][]*Pin),
		AllPins:    make([]*Pin, 0),
	}
}

// RegisterPin adds a pin to the registry.
func (r *PinRegistry) RegisterPin(pin *Pin) {
	if pin == nil || pin.SemanticType == "" {
		return
	}

	r.AllPins = append(r.AllPins, pin)

	if pin.IsInput() {
		r.InputPins[pin.SemanticType] = append(r.InputPins[pin.SemanticType], pin)
	} else {
		r.OutputPins[pin.SemanticType] = append(r.OutputPins[pin.SemanticType], pin)
	}
}

// GetProducers returns all output pins that produce the given semantic type.
func (r *PinRegistry) GetProducers(semanticType SemanticType) []*Pin {
	return r.OutputPins[semanticType]
}

// GetConsumers returns all input pins that consume the given semantic type.
func (r *PinRegistry) GetConsumers(semanticType SemanticType) []*Pin {
	return r.InputPins[semanticType]
}

// GetConnections returns all possible connections (producer -> consumer pairs).
func (r *PinRegistry) GetConnections() []PinConnection {
	var connections []PinConnection

	for semanticType, producers := range r.OutputPins {
		consumers := r.InputPins[semanticType]
		if len(consumers) == 0 {
			continue
		}

		for _, producer := range producers {
			for _, consumer := range consumers {
				// Don't connect pins from the same endpoint
				if producer.Location.EndpointPath == consumer.Location.EndpointPath &&
					producer.Location.EndpointMethod == consumer.Location.EndpointMethod {
					continue
				}

				connections = append(connections, PinConnection{
					Producer: producer,
					Consumer: consumer,
				})
			}
		}
	}

	return connections
}

// GetUnconnectedInputs returns input pins that have no matching producers.
func (r *PinRegistry) GetUnconnectedInputs() []*Pin {
	var unconnected []*Pin

	for semanticType, inputs := range r.InputPins {
		if len(r.OutputPins[semanticType]) == 0 {
			unconnected = append(unconnected, inputs...)
		}
	}

	return unconnected
}

// GetUnconnectedOutputs returns output pins that have no matching consumers.
func (r *PinRegistry) GetUnconnectedOutputs() []*Pin {
	var unconnected []*Pin

	for semanticType, outputs := range r.OutputPins {
		if len(r.InputPins[semanticType]) == 0 {
			unconnected = append(unconnected, outputs...)
		}
	}

	return unconnected
}

// Stats returns statistics about the pin registry.
func (r *PinRegistry) Stats() PinRegistryStats {
	stats := PinRegistryStats{
		TotalPins:        len(r.AllPins),
		SemanticTypes:    make(map[SemanticType]int),
		InputsByCategory: make(map[string]int),
	}

	for _, pin := range r.AllPins {
		if pin.IsInput() {
			stats.TotalInputPins++
		} else {
			stats.TotalOutputPins++
		}

		if pin.SemanticType != "" && pin.SemanticType != UnknownSemanticType {
			stats.SemanticTypes[pin.SemanticType]++
			stats.InputsByCategory[pin.SemanticType.Category()]++
		} else {
			stats.UnknownTypePins++
		}

		if pin.IsHighConfidence() {
			stats.HighConfidencePins++
		}
	}

	return stats
}

// PinConnection represents a connection between a producer and consumer pin.
type PinConnection struct {
	Producer *Pin
	Consumer *Pin
}

// PinRegistryStats holds statistics about a pin registry.
type PinRegistryStats struct {
	TotalPins          int
	TotalInputPins     int
	TotalOutputPins    int
	UnknownTypePins    int
	HighConfidencePins int
	SemanticTypes      map[SemanticType]int
	InputsByCategory   map[string]int
}
