# Endure [currently in beta]
<p align="center">
	<a href="https://pkg.go.dev/github.com/spiral/Endure?tab=doc"><img src="https://godoc.org/github.com/spiral/Endure?status.svg"></a>
	<a href="https://github.com/spiral/Endure/actions"><img src="https://github.com/spiral/Endure/workflows/CI/badge.svg" alt=""></a>
	<a href="https://goreportcard.com/report/github.com/spiral/Endure"><img src="https://goreportcard.com/badge/github.com/spiral/Endure"></a>
	<a href="https://codecov.io/gh/spiral/Endure/"><img src="https://codecov.io/gh/spiral/Endure/branch/master/graph/badge.svg"></a>
	<a href="https://discord.gg/TFeEmCs"><img src="https://img.shields.io/badge/discord-chat-magenta.svg"></a>
</p>

Endure is an open-source (MIT licensed) plugin container.

<h2>Features</h2>

- Supports structs and interfaces (see examples)
- Use graph to topologically sort, run, stop and restart dependent plugins
- Algorithms used: graph and double-linked list
- Support easy to add Middleware plugins
- Error reporting
- Automatically restart failing vertices


<h2>Installation</h2>  

```go
go get -u github.com/spiral/Endure
```  


<h2>Why?</h2>  

Imagine you have an application in which you want to implement plugin system. These plugins can depend on each other (via interfaces or directly).
For example, we have 3 plugins: HTTP (to communicate with world), DB (to save the world) and logger (to see the progress).  
In this particular case, we can't start HTTP before we start all other parts. Also, we have to initialize logger first, because all parts of our system needs logger. All you need to do in `Endure` is to pass `HTTP`, `DB` and `logger` structs to the `Endure` and implement `Endure` interface. So, the dependency graph will be the following:
<p align="left">
  <img src="https://github.com/spiral/endure/blob/master/images/graph.png" width="300" height="250" />
</p>

Next we need to start all part:
```go
errCh, err := container.Serve()
```
`errCh` is the channel with error from the all `Vertices`. You can identify vertex by `vertexID` presented in errCh struct.
And then just process the events from the `errCh`:
```go
	for {
		select {
		case e := <-errCh:
			println(e.Error.Err.Error()) // just print the error
			er := container.Stop()
			if er != nil {
				panic(er)
			}
			return
		}
	}
```
Also `Endure` will take care of restar failing vertices (structs, HTTP for example) with exponential backoff mechanism, star in topological order (`logger` -> `DB` -> `HTTP`) and stop in reverse-topological order automatically.


<h2>Endure main interface</h2>  

```go
package sample

type (
	// used to gracefully stop and configure the plugins
	Graceful interface {
		// Configure is used when we need to make preparation and wait for all services till Serve
		Configure() error
		// Close frees resources allocated by the service
		Close() error
	}
	// this is the main service interface with should implement every plugin
	Service interface {
		// Serve
		Serve() chan error
		// Stop
		Stop() error
	}

	// Name of the service
	Named interface {
		Name() string
	}

	// Provider declares the ability to provide service edges of declared types.
	Provider interface {
		Provides() []interface{}
	}

	// Depender declares the ability to accept the plugins which match the provided method signature.
	Depender interface {
		Depends() []interface{}
	}
)  
```
Order is the following:
1. `Init() error` - mandatory to implement. In your structure (which you pass to Endure), you should have this method as receiver. It can accept as parameter any passed to the `Endure` structure (see sample) or interface (with limitations).  
2. `Graceful` - optional to implement. Used to configure a vertex before invoking `Serve` method. Has the `Confugure` method which will be invoked after `Init` and `Close` which will be invoked after `Stop` to free some resources for example.
3. `Service` - mandatory to implement. Has 2 main methods - `Serve` which should return initialized golang channel with errors, and `Stop` to stop the shutdown the Endure.
4. `Provider` - optional to implement. Used to provide some dependency if you need to extend your struct.
5. `Depender` - optional to implement. Used to mark structure (vertex) as some struct dependency. It can accept interfaces which implement caller.
6. `Named` - optional to implement. That is a special kind of interface to provide the name of the struct (plugin, vertex) to the caller. Useful in logger to know friendly plugin name.
