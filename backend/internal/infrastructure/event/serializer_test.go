package event

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serializerTestEvent is a test event for serializer tests
type serializerTestEvent struct {
	shared.BaseDomainEvent
	Data    string `json:"data"`
	Counter int    `json:"counter"`
}

func newSerializerTestEvent() *serializerTestEvent {
	return &serializerTestEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("SerializerTestEvent", "TestAggregate", uuid.New(), uuid.New()),
		Data:            "test data",
		Counter:         42,
	}
}

func TestEventSerializer_Register(t *testing.T) {
	serializer := NewEventSerializer()

	serializer.Register("SerializerTestEvent", &serializerTestEvent{})

	assert.True(t, serializer.IsRegistered("SerializerTestEvent"))
	assert.False(t, serializer.IsRegistered("UnknownEvent"))
}

func TestEventSerializer_RegisteredTypes(t *testing.T) {
	serializer := NewEventSerializer()

	serializer.Register("Event1", &serializerTestEvent{})
	serializer.Register("Event2", &serializerTestEvent{})

	types := serializer.RegisteredTypes()
	assert.Len(t, types, 2)
	assert.Contains(t, types, "Event1")
	assert.Contains(t, types, "Event2")
}

func TestEventSerializer_Serialize(t *testing.T) {
	serializer := NewEventSerializer()
	event := newSerializerTestEvent()

	data, err := serializer.Serialize(event)

	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), `"data":"test data"`)
	assert.Contains(t, string(data), `"counter":42`)
}

func TestEventSerializer_Deserialize(t *testing.T) {
	serializer := NewEventSerializer()
	serializer.Register("SerializerTestEvent", &serializerTestEvent{})

	original := newSerializerTestEvent()
	data, err := serializer.Serialize(original)
	require.NoError(t, err)

	deserialized, err := serializer.Deserialize("SerializerTestEvent", data)
	require.NoError(t, err)

	event, ok := deserialized.(*serializerTestEvent)
	require.True(t, ok)
	assert.Equal(t, original.EventType(), event.EventType())
	assert.Equal(t, original.Data, event.Data)
	assert.Equal(t, original.Counter, event.Counter)
}

func TestEventSerializer_Deserialize_UnknownType(t *testing.T) {
	serializer := NewEventSerializer()

	_, err := serializer.Deserialize("UnknownEvent", []byte(`{}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type")
}

func TestEventSerializer_Deserialize_InvalidJSON(t *testing.T) {
	serializer := NewEventSerializer()
	serializer.Register("SerializerTestEvent", &serializerTestEvent{})

	_, err := serializer.Deserialize("SerializerTestEvent", []byte(`invalid json`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

func TestEventSerializer_RoundTrip_PreservesAllFields(t *testing.T) {
	serializer := NewEventSerializer()
	serializer.Register("SerializerTestEvent", &serializerTestEvent{})

	tenantID := uuid.New()
	aggregateID := uuid.New()
	original := &serializerTestEvent{
		BaseDomainEvent: shared.BaseDomainEvent{
			ID:            uuid.New(),
			Type:          "SerializerTestEvent",
			Timestamp:     time.Now().Truncate(time.Second),
			AggID:         aggregateID,
			AggType:       "TestAggregate",
			TenantIDValue: tenantID,
		},
		Data:    "important data",
		Counter: 99,
	}

	data, err := serializer.Serialize(original)
	require.NoError(t, err)

	deserialized, err := serializer.Deserialize("SerializerTestEvent", data)
	require.NoError(t, err)

	event := deserialized.(*serializerTestEvent)
	assert.Equal(t, original.EventID(), event.EventID())
	assert.Equal(t, original.EventType(), event.EventType())
	assert.Equal(t, original.AggregateID(), event.AggregateID())
	assert.Equal(t, original.AggregateType(), event.AggregateType())
	assert.Equal(t, original.TenantID(), event.TenantID())
	assert.Equal(t, original.Data, event.Data)
	assert.Equal(t, original.Counter, event.Counter)
}
