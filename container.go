package cascade

// InitMethodName is the function name for the reflection
const InitMethodName = "Init"

// ConfigureMethodName
const ConfigureMethodName = "Configure"

// CloseMethodName
const CloseMethodName = "Close"

// ServeMethodName
const ServeMethodName = "Serve"

// Stop is the function name for the reflection to Stop the service
const StopMethodName = "Stop"

// TODO interface?
type Result struct {
	Err      error
	Code     int
	VertexID string
}

type result struct {
	// error channel from vertex
	errCh chan error
	// unique vertex id
	vertexId string
	// signal to the vertex goroutine to exit
	exit chan struct{}
}

type Error struct {
	Err   error
	Code  int
	Stack []uint
}

type (
	// TODO namings
	Graceful interface {
		// Configure is used when we need to make preparation and wait for all services till Serve
		Configure() error
		// Close frees resources allocated by the service
		Close() error
	}
	Service interface {
		// Serve
		Serve() chan error
		// Stop
		Stop() error
	}

	Container interface {
		Serve() <-chan *Result
		Close() error
		Register(service interface{}) error
		Init() error
	}

	// Provider declares the ability to provide service edges of declared types.
	Provider interface {
		Provides() []interface{}
	}

	// Register declares the ability to accept the plugins which match the provided method signature.
	Register interface {
		Depends() []interface{}
	}
)
