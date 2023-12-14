package bus

import (
	"database/sql/driver"

	"github.com/YuukanOO/seelf/pkg/storage"
)

type UnmarshalFunc func(string, string) (Request, error)

// Contains available message name -> unmarshal function
var marshallable map[string]UnmarshalFunc = make(map[string]UnmarshalFunc)

// Rebuild a message from a given name and serialized data.
func UnmarshalMessage(name, data string) (Request, error) {
	fn, found := marshallable[name]

	if !found {
		return nil, storage.ErrCouldNotUnmarshalGivenType
	}

	return fn(name, data)
}

// Marshal the given message
func MarshalMessage[T Request](msg T) (driver.Value, error) { return storage.ValueJSON(msg) }

// Register given message type for marshalling which will make it marshallable
// and unmarshallable to and from JSON. This is needed when persisting scheduled jobs.
func RegisterForMarshalling[T Request]() {
	var msg T
	marshallable[msg.Name_()] = func(name, data string) (Request, error) {
		var out T
		return out, storage.ScanJSON(data, &out)
	}
}
