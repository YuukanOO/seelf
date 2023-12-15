package bus

import (
	"database/sql/driver"

	"github.com/YuukanOO/seelf/pkg/storage"
)

// Contains available message name -> unmarshal function
var Marshallable = storage.NewDiscriminatedMapper(func(r Request) string { return r.Name_() })

// Marshal the given message. Simple helper func, it justs call storage.ValueJSON.
// To rehydrate a message, just call bus.Marshallable.From.
func MarshalMessage[T Request](msg T) (driver.Value, error) { return storage.ValueJSON(msg) }

// Register given message type for marshalling which will make it marshallable
// and unmarshallable to and from JSON. This is needed when persisting scheduled jobs.
// Simple helper function to register the type on the Marshallable discriminated mapper
// using storage.ScanJSON.
func RegisterForMarshalling[T Request]() {
	var msg T

	Marshallable.Register(msg, func(data string) (Request, error) {
		var out T
		return out, storage.ScanJSON(data, &out)
	})
}
