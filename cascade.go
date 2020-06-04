package cascade

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/spiral/cascade/structures"
)

const Init = "Init"

type Cascade struct {
	deps map[string][]structures.Dep

	providers     map[reflect.Type]entry
	depends       map[reflect.Type][]entry
	servicesGraph *structures.Graph
}

type entry struct {
	name   string
	vertex interface{}
}

func NewContainer() *Cascade {
	return &Cascade{
		deps:          make(map[string][]structures.Dep),
		depends:       make(map[reflect.Type][]entry),
		providers:     make(map[reflect.Type]entry),
		servicesGraph: structures.NewAL(),
	}
}

// Register depends the dependencies
// name is a name of the dependency, for example - S2
// vertex is a value -> pointer to the structure
func (c *Cascade) Register(name string, vertex interface{}) error {
	if c.servicesGraph.Has(name) {
		return fmt.Errorf("vertex `%s` already exists", name)
	}

	// just push the vertex
	// here we can append in future some meta information
	c.servicesGraph.AddVertex(name, vertex, reflect.TypeOf(vertex).String())

	if provider, ok := vertex.(Provider); ok {
		for _, fn := range provider.Provides() {
			ret, err := returnType(fn)
			if err != nil {
				// todo: delete vertex
				return err
			}
			// save providers
			c.providers[ret] = entry{name: name, vertex: fn}
		}
	}

	if register, ok := vertex.(Register); ok {
		for _, fn := range register.Depends() {
			// what type it might depend on?
			argsTypes, err := argType(fn)
			if err != nil {
				// todo: delete vertex
				return err
			}

			// empty Depends, show warning and continue
			if len(argsTypes) == 0 {
				fmt.Printf("%s must accept exactly one argument", fn)
			}

			if len(argsTypes) > 0 {
				for _, at := range argsTypes {
					// if we found, that some structure depends on some type
					// we also save it in the `depends` section
					// name s1 (for example)
					// vertex - S4 func
					c.depends[at] = append(c.depends[at], entry{name: name, vertex: fn})
				}
			} else {
				// todo temporary
				panic("argsTypes less than 0")
			}
		}
	}

	return nil
}

// Init container and all service edges.
func (c *Cascade) Init() error {
	// traverse the graph
	if err := c.calculateDependencies(); err != nil {
		return err
	}

	c.flattenSimpleGraph()
	s := c.topologicalSort()
	fmt.Println(s)

	return nil
}

// calculateDependencies calculates simple graph for the dependencies
func (c *Cascade) calculateDependencies() error {
	// name for example S2
	for name, vrtx := range c.servicesGraph.Vertices {
		init, ok := reflect.TypeOf(vrtx.Value).MethodByName(Init)
		if !ok {
			continue
		}

		// S2 init args
		initArgs, err := functionParameters(init)
		if err != nil {
			return err
		}

		// iterate over all function parameters
		for _, initArg := range initArgs {
			for id, vertex := range c.servicesGraph.Vertices {
				if id == name {
					continue
				}

				initArgTr := removePointerAsterisk(initArg.String())
				vertexTypeTr := removePointerAsterisk(reflect.TypeOf(vertex.Value).String())

				// guess, the types are the same type
				if initArgTr == vertexTypeTr {
					c.servicesGraph.AddEdge(name, id)
				}
			}

			// provides type (DB for example)
			// and entry for that type
			for t, e := range c.providers {
				provider := removePointerAsterisk(t.String())

				if provider == initArg.String() {
					if c.servicesGraph.Has(name) == false {
						c.servicesGraph.AddEdge(name, e.name)
					}
				}
			}
		}

		// second round of the dependencies search
		// via the depends
		// in the tests, S1 depends on the S4 and S2 on the S4 via the Depends interface
		// a lot of stupid allocations here, needed to be optimized in the future
		for rflType, slice := range c.depends {
			// check if we iterate over the needed type
			if removePointerAsterisk(rflType.String()) == removePointerAsterisk(vrtx.Meta.RawPackage) {
				for _, entry := range slice {
					// rflType --> S4
					// in slice s1, s2

					// guard here
					entryType, err := argType(entry.vertex)
					if err != nil {
						return err
					}
					if len(entryType) > 0 {
						for _, et := range entryType {
							// s3:[s4 s2 s2] TODO
							if removePointerAsterisk(et.String()) == removePointerAsterisk(rflType.String()) {
								c.servicesGraph.AddEdge(entry.name, name)
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// flattenSimpleGraph flattens the graph, making the following structure
// S1 -> S2 | S2 -> S4 | S3 -> S2 | S4 |
// S1 -> S4 |          | S3 -> S4 |    |
//
func (c *Cascade) flattenSimpleGraph() {
	for key, edge := range c.servicesGraph.Edges {
		if len(edge) == 0 {
			// no dependencies, just add the standalone
			d := structures.Dep{
				Id: key,
				D:  nil,
			}

			c.deps[key] = append(c.deps[key], d)
		}
		for _, e := range edge {
			d := structures.Dep{
				Id: key,
				D:  e,
			}

			c.deps[key] = append(c.deps[key], d)
		}
	}
}

func (c *Cascade) topologicalSort() []string {
	ids := make([]string, 0)
	for k, _ := range c.servicesGraph.Vertices {
		ids = append(ids, k)
	}

	gr := structures.NewDepsGraph(ids)

	for id, dep := range c.deps {
		for _, v := range dep {
			if v.D == nil {
				continue
			}
			gr.AddDep(id, v.D.(string))
		}
	}

	return gr.Order()
}

func removePointerAsterisk(s string) string {
	return strings.Trim(s, "*")
}
