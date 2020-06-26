package cascade

import (
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/spiral/cascade/structures"
)

type Cascade struct {
	// Dependency graph
	graph *structures.Graph
	// DLL used as run list to run in order
	runList *structures.DoublyLinkedList
	// logger
	logger zerolog.Logger
	// OPTIONS
	retryOnFail  bool
	numOfRetries int

	rwMutex *sync.RWMutex

	results map[string]*result

	failProcessor func(k *Result) chan *Result
}

// Level defines log levels.
type Level int8

const (
	// DebugLevel defines debug log level.
	DebugLevel Level = iota
	// InfoLevel defines info log level.
	InfoLevel
	// WarnLevel defines warn log level.
	WarnLevel
	// ErrorLevel defines error log level.
	ErrorLevel
	// FatalLevel defines fatal log level.
	FatalLevel
	// PanicLevel defines panic log level.
	PanicLevel
	// NoLevel defines an absent log level.
	NoLevel
	// Disabled disables the logger.
	Disabled

	// TraceLevel defines trace log level.
	TraceLevel Level = -1
)

type Options func(cascade *Cascade)

////////////////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////// PUBLIC ////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////////////

/* Input parameters: logLevel
-1 is the most informative level - TraceLevel
0 - DebugLevel defines debug log level
1 - InfoLevel defines info log level.
2 - WarnLevel defines warn log level.
3 - ErrorLevel defines error log level.
4 - FatalLevel defines fatal log level.
5 - PanicLevel defines panic log level.
6 - NoLevel defines an absent log level.
7 - Disabled disables the logger.
see the cascade.Level
*/
func NewContainer(logLevel Level, options ...Options) (*Cascade, error) {
	c := &Cascade{
		rwMutex: &sync.RWMutex{},
	}
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	switch logLevel {
	case DebugLevel:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case InfoLevel:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case WarnLevel:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case ErrorLevel:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case FatalLevel:
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case PanicLevel:
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	case NoLevel:
		zerolog.SetGlobalLevel(zerolog.NoLevel)
	case Disabled:
		zerolog.SetGlobalLevel(zerolog.Disabled)
	case -1: // TraceLevel
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	for _, option := range options {
		option(c)
	}

	c.graph = structures.NewGraph()
	c.runList = structures.NewDoublyLinkedList()
	c.logger = logger
	c.results = make(map[string]*result)
	//c.failProcessor =

	//c := make(chan Time, 1)
	//t := &Timer{
	//	C: c,
	//	r: runtimeTimer{
	//		when: when(d),
	//		f:    sendTime,
	//		arg:  c,
	//	},
	//}
	//startTimer(&t.r)
	//return t

	return c, nil
}

func RetryOnFail(set bool) Options {
	return func(cascade *Cascade) {
		cascade.retryOnFail = set
		// default value
		cascade.numOfRetries = 5
	}
}

// Register depends the dependencies
// name is a name of the dependency, for example - S2
// vertex is a value -> pointer to the structure
func (c *Cascade) Register(vertex interface{}) error {
	t := reflect.TypeOf(vertex)
	vertexID := removePointerAsterisk(t.String())

	ok := t.Implements(reflect.TypeOf((*Service)(nil)).Elem())
	if !ok {
		return typeNotImplementError
	}

	/* Register the type
	Information we know at this step is:
	1. VertexId
	2. Vertex structure value (interface)
	And we fill vertex with this information
	*/
	err := c.register(vertexID, vertex)
	if err != nil {
		return err
	}

	/* Add the types, which (if) current vertex provides
	Information we know at this step is:
	1. VertexId
	2. Vertex structure value (interface)
	3. Provided type
	4. Provided type String name
	We add 3 and 4 points to the Vertex
	*/
	err = c.addProviders(vertexID, vertex)
	if err != nil {
		return err
	}

	c.logger.Info().Msgf("registered type: %s", t.String())

	return nil
}

// Init container and all service edges.
func (c *Cascade) Init() error {
	// traverse the graph
	if err := c.addEdges(); err != nil {
		return err
	}

	// we should buld init list in the reverse order
	// TODO return cycle error
	sortedVertices := structures.OldTopologicalSort(c.graph.Vertices)

	// TODO properly handle the len of the sorted vertices
	c.runList.SetHead(&structures.DllNode{
		Vertex: sortedVertices[0]})

	// TODO what if sortedVertices will contain only 1 node (len(sortedVertices) - 2 will panic)
	for i := 1; i < len(sortedVertices); i++ {
		c.runList.Push(sortedVertices[i])
	}

	head := c.runList.Head

	err := c.init(head)
	if err != nil {
		c.logger.
			Err(err).
			Stack().
			Msg("error during the init")
		return err
	}
	return nil
}

func (c *Cascade) Configure() error {
	return nil
}

func (c *Cascade) Close() error {
	return nil
}

func (c *Cascade) Serve() <-chan *Result {
	n := c.runList.Head

	err := c.serveVertices(n)
	if err != nil {
		panic(err)
	}



	if c.retryOnFail {
		out := make(chan *Result)

		for k, v := range c.results {
			go func(vertexId string, res *result) {
				for {
					time.Sleep(time.Second * 1)
					select {
					case e := <-res.errCh:
						if e != nil {
							println("EROOOOOOOOOOOOOOOR OCCURRED" + e.Error())
							out <- &Result{
								Err:      e,
								Code:     0,
								VertexID: vertexId,
							}

							err := c.defaultPoller(vertexId)
							if err != nil {
								panic(err)
							}
						}

					}
				}
			}(k, v)
		}
		return out
	}

	// read message
	// do retry
	// clone the message and re-send it to the receiver
	r := make([]*result, 0, 5)
	for _, v := range c.results {
		r = append(r, v)
	}
	return merge(r)
}

func (c *Cascade) defaultPoller(vId string) error {
	// get the vertex
	// calculate dependencies
	// close/stop affected vertices
	// build new topologically sorted graph and new run-list
	// re-serve and connect messages to the clonedRes channel

	vertex := c.graph.GetVertex(vId)

	vertices := c.resetVertices(vertex)

	sorted := structures.TopologicalSort(vertices)

	affectedRunList := structures.NewDoublyLinkedList()
	// TODO properly handle the len of the sorted vertices
	affectedRunList.SetHead(&structures.DllNode{
		Vertex: sorted[len(sorted)-1]})

	// TODO what if sortedVertices will contain only 1 node (len(sortedVertices) - 2 will panic)
	for i := len(sorted) - 2; i >= 0; i-- {
		affectedRunList.Push(sorted[i])
	}

	nodes := affectedRunList.Head

	in := make([]reflect.Value, 0, 1)
	// add service itself
	in = append(in, reflect.ValueOf(nodes.Vertex.Iface))
	cNodes := nodes
	for cNodes != nil {
		err := c.internalStop(cNodes)
		if err != nil {
			// TODO do not return until finished
			// just log the errors
			// stack it in slice and if slice is not empty, print it ??
			c.logger.Err(err).Stack().Msg("error occurred during the services stopping")
		}

		// prev DLL node
		cNodes = cNodes.Next
	}

	nn := nodes
	err := c.init(nn)
	if err != nil {
		c.logger.
			Err(err).
			Stack().
			Msg("error during the retry init")
		return nil
	}

	err = c.serveVertices(nodes)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cascade) Stop() error {
	n := c.runList.Head

	for n != nil {
		err := c.internalStop(n)
		if err != nil {
			// TODO do not return until finished
			// just log the errors
			// stack it in slice and if slice is not empty, print it ??
			c.logger.Err(err).Stack().Msg("error occurred during the services stopping")
		}

		// prev DLL node
		n = n.Next
	}
	return nil
}

func (c *Cascade) resetVertices(vertex *structures.Vertex) []*structures.Vertex {
	// restore number of dependencies for the root
	vertex.NumOfDeps = len(vertex.Dependencies)
	vertex.Visiting = false
	vertex.Visited = false
	vertices := make([]*structures.Vertex, 0, 5)
	vertices = append(vertices, vertex)

	tmp := make(map[string]*structures.Vertex)

	c.dfs(vertex.Dependencies, tmp)

	for _, v := range tmp {
		vertices = append(vertices, v)
	}
	return vertices
}

func (c *Cascade) dfs(deps []*structures.Vertex, tmp map[string]*structures.Vertex) {
	for i := 0; i < len(deps); i++ {
		deps[i].Visited = false
		deps[i].Visiting = false
		deps[i].NumOfDeps = len(deps)
		tmp[deps[i].Id] = deps[i]
		c.dfs(deps[i].Dependencies, tmp)
	}

}

////////////////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////// PRIVATE ///////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////////////

func (c *Cascade) register(name string, vertex interface{}) error {
	// check the vertex
	if c.graph.HasVertex(name) {
		return vertexAlreadyExists(name)
	}

	// just push the vertex
	// here we can append in future some meta information
	c.graph.AddVertex(name, vertex, structures.Meta{})
	return nil
}

/*
Traverse the DLL in the forward direction

*/
func (c *Cascade) init(n *structures.DllNode) error {
	// traverse the dll
	for n != nil {
		// we already checked the Interface satisfaction
		// at this step absence of Init() is impossible
		init, _ := reflect.TypeOf(n.Vertex.Iface).MethodByName(InitMethodName)

		err := c.funcCall(init, n)
		if err != nil {
			c.logger.
				Err(err).
				Stack().Str("vertexID", n.Vertex.Id).
				Msg("error occurred while calling a function")
			return err
		}

		// next DLL node
		n = n.Next
	}

	return nil
}
