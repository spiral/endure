package cascade

import (
	"errors"
	"reflect"

	"github.com/spiral/cascade/structures"
)

func (c *Cascade) funcCall(init reflect.Method, n *structures.DllNode) error {
	in := c.getInitValues(n)

	// Iterate over dependencies
	// And search in Vertices for the provided types
	ret := init.Func.Call(in)
	rErr := ret[0].Interface()
	if rErr != nil {
		if e, ok := rErr.(error); ok && e != nil {
			c.logger.Err(e)
			return e
		} else {
			return unknownErrorOccurred
		}
	}

	// just to be safe here
	if len(in) > 0 {
		/*
			n.Vertex.AddValue
			1. removePointerAsterisk to have uniform way of adding and searching the function args
		*/
		err := n.Vertex.AddValue(removePointerAsterisk(in[0].Type().String()), in[0], isReference(in[0].Type()))
		if err != nil {
			return err
		}
		c.logger.Info().
			Str("vertexID", n.Vertex.Id).
			Str("IN parameter", in[0].Type().String()).
			Msg("value added successfully")
	} else {
		panic("len in less than 2")
	}

	err := c.traverseCallProvider(n, []reflect.Value{reflect.ValueOf(n.Vertex.Iface)})
	if err != nil {
		return err
	}

	err = c.traverseCallRegisters(n)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cascade) traverseCallRegisters(n *structures.DllNode) error {
	inReg := make([]reflect.Value, 0, 1)

	// add service itself
	inReg = append(inReg, reflect.ValueOf(n.Vertex.Iface))

	// add dependencies
	if len(n.Vertex.Meta.DepsList) > 0 {
		for i := 0; i < len(n.Vertex.Meta.DepsList); i++ {
			depId := n.Vertex.Meta.DepsList[i].Name
			v := c.graph.FindProvider(depId)

			for k, val := range v.Provides {
				if k == depId {
					// value - reference and init dep also reference
					if *val.IsReference == *n.Vertex.Meta.DepsList[i].IsReference {
						inReg = append(inReg, *val.Value)
					} else if *val.IsReference {
						// same type, but difference in the refs
						// Init needs to be a value
						// But Vertex provided reference

						inReg = append(inReg, val.Value.Elem())
					} else if !*val.IsReference {
						// vice versa
						// Vertex provided value
						// but Init needs to be a reference
						if val.Value.CanAddr() {
							inReg = append(inReg, val.Value.Addr())
						} else {
							c.logger.Warn().Str("type", val.Value.Type().String()).Msgf("value is not addressible. TIP: consider to return a pointer from %s", val.Value.Type())
							c.logger.Warn().Msgf("making a fresh pointer")

							nt := reflect.New(val.Value.Type())
							inReg = append(inReg, nt)
						}
					}
				}
			}
		}
	}

	//type implements Register interface
	if reflect.TypeOf(n.Vertex.Iface).Implements(reflect.TypeOf((*Register)(nil)).Elem()) {
		// if type implements Register() it should has FnsProviderToInvoke
		if n.Vertex.Meta.DepsList != nil {
			for i := 0; i < len(n.Vertex.Meta.FnsRegisterToInvoke); i++ {
				m, ok := reflect.TypeOf(n.Vertex.Iface).MethodByName(n.Vertex.Meta.FnsRegisterToInvoke[i])
				if !ok {
					panic("method Provides should be")
				}

				ret := m.Func.Call(inReg)
				// handle error
				if len(ret) > 0 {
					rErr := ret[0].Interface()
					if rErr != nil {
						if e, ok := rErr.(error); ok && e != nil {
							c.logger.Err(e).Msg("error occurred during the Registers invocation")
							return e
						} else {
							return unknownErrorOccurred
						}
					}
				} else {
					return errors.New("register should return Value and error types")
				}
			}
		}
	}
	return nil
}

func (c *Cascade) traverseCallProvider(n *structures.DllNode, in []reflect.Value) error {
	// type implements Provider interface
	if reflect.TypeOf(n.Vertex.Iface).Implements(reflect.TypeOf((*Provider)(nil)).Elem()) {
		// if type implements Provider() it should has FnsProviderToInvoke
		if n.Vertex.Meta.FnsProviderToInvoke != nil {
			for i := 0; i < len(n.Vertex.Meta.FnsProviderToInvoke); i++ {
				m, ok := reflect.TypeOf(n.Vertex.Iface).MethodByName(n.Vertex.Meta.FnsProviderToInvoke[i])
				if !ok {
					panic("method Provides should be")
				}

				ret := m.Func.Call(in)
				// handle error
				if len(ret) > 1 {
					rErr := ret[1].Interface()
					if rErr != nil {
						if e, ok := rErr.(error); ok && e != nil {
							c.logger.Err(e).Msg("error occurred in the traverseCallProvider")
							return e
						} else {
							return unknownErrorOccurred
						}
					}

					err := n.Vertex.AddValue(removePointerAsterisk(ret[0].Type().String()), ret[0], isReference(ret[0].Type()))
					if err != nil {
						return err
					}
				} else {
					return errors.New("provider should return Value and error types")
				}
			}
		}
	}
	return nil
}

func (c *Cascade) getInitValues(n *structures.DllNode) []reflect.Value {
	in := make([]reflect.Value, 0, 1)

	// add service itself
	in = append(in, reflect.ValueOf(n.Vertex.Iface))

	// add dependencies
	if len(n.Vertex.Meta.InitDepsList) > 0 {
		for i := 0; i < len(n.Vertex.Meta.InitDepsList); i++ {
			depId := n.Vertex.Meta.InitDepsList[i].Name
			v := c.graph.FindProvider(depId)

			for k, val := range v.Provides {
				if k == depId {
					// value - reference and init dep also reference
					if *val.IsReference == *n.Vertex.Meta.InitDepsList[i].IsReference {
						in = append(in, *val.Value)
					} else if *val.IsReference {
						// same type, but difference in the refs
						// Init needs to be a value
						// But Vertex provided reference

						in = append(in, val.Value.Elem())
					} else if !*val.IsReference {
						// vice versa
						// Vertex provided value
						// but Init needs to be a reference
						if val.Value.CanAddr() {
							in = append(in, val.Value.Addr())
						} else {
							c.logger.Warn().Str("type", val.Value.Type().String()).Msgf("value is not addressible. TIP: consider to return a pointer from %s", val.Value.Type())
							c.logger.Warn().Msgf("making a fresh pointer")
							nt := reflect.New(val.Value.Type())
							in = append(in, nt)
						}
					}
				}
			}
		}
	}
	return in
}

/*
Algorithm is the following (all steps executing in the topological order):
1. Call Configure() on all services -- OPTIONAL
2. Call Serve() on all services --     MUST
3. Call Stop() on all services --      MUST
4. Call Clear() on a services, which implements this interface -- OPTIONAL
*/
// call configure on the node

func (c *Cascade) serveVertices(n *structures.DllNode) error {
	// TODO len of DDLNodes
	//out := make([]*result, 0, 5)
	nCopy := n
	// handle all configure
	for nCopy != nil {
		in := make([]reflect.Value, 0, 1)

		// add service itself
		in = append(in, reflect.ValueOf(nCopy.Vertex.Iface))
		//var res Result
		if reflect.TypeOf(nCopy.Vertex.Iface).Implements(reflect.TypeOf((*Graceful)(nil)).Elem()) {
			// call configure
			//out = append(out, c.call(nCopy, in, ConfigureMethodName))
			err := c.call(nCopy, in, ConfigureMethodName)
			if err != nil {
				return err
			}
		}

		nCopy = nCopy.Next
	}

	// reset the list
	nCopy = n
	// and handle all serve
	for nCopy != nil {
		in := make([]reflect.Value, 0, 1)

		// add service itself
		in = append(in, reflect.ValueOf(nCopy.Vertex.Iface))

		// call serve
		//out = append(out, c.call(nCopy, in, ServeMethodName))
		err := c.call(nCopy, in, ServeMethodName)
		if err != nil {
			return err
		}
		nCopy = nCopy.Next
	}
	return nil
}

func (c *Cascade) call(n *structures.DllNode, in []reflect.Value, methodName string) error {
	// Call Configure() method, which returns only error (or nil)
	m, _ := reflect.TypeOf(n.Vertex.Iface).MethodByName(methodName)
	ret := m.Func.Call(in)
	res := ret[0].Interface()
	if res != nil {
		if e, ok := res.(chan error); ok && e != nil {
			switch methodName {
			case ServeMethodName:
				c.results[n.Vertex.Id] = &result{
					errCh:    e,
					vertexId: n.Vertex.Id,
				}
			case ConfigureMethodName:

			}

			return nil
			// return the result
			//return &result{
			//	errCh:    e,
			//	vertexId: n.Vertex.Id,
			//}
		}
	}
	//r := &result{
	//	errCh:    make(chan error, 1),
	//	vertexId: n.Vertex.Id,
	//}
	//r.errCh <- unknownErrorOccurred
	//return r
	return unknownErrorOccurred
}

func (c *Cascade) internalStop(n *structures.DllNode) error {
	in := make([]reflect.Value, 0, 1)

	// add service itself
	in = append(in, reflect.ValueOf(n.Vertex.Iface))

	err := c.stop(n, in)
	if err != nil {
		c.logger.Err(err).Stack().Msg("error occurred during the stop")
	}

	if reflect.TypeOf(n.Vertex.Iface).Implements(reflect.TypeOf((*Graceful)(nil)).Elem()) {
		err = c.close(n, in)
		if err != nil {
			c.logger.Err(err).Stack().Msg("error occurred during the close")
		}
	}

	return nil
}

func (c *Cascade) stop(n *structures.DllNode, in []reflect.Value) error {
	// Call Stop() method, which returns only error (or nil)
	m, _ := reflect.TypeOf(n.Vertex.Iface).MethodByName(StopMethodName)
	ret := m.Func.Call(in)
	rErr := ret[0].Interface()
	if rErr != nil {
		if e, ok := rErr.(error); ok && e != nil {
			return e
		} else {
			return unknownErrorOccurred
		}
	}
	return nil
}

// TODO add stack to the all of the log events
func (c *Cascade) close(n *structures.DllNode, in []reflect.Value) error {
	// Call Close() method, which returns only error (or nil)
	m, _ := reflect.TypeOf(n.Vertex.Iface).MethodByName(CloseMethodName)
	ret := m.Func.Call(in)
	rErr := ret[0].Interface()
	if rErr != nil {
		if e, ok := rErr.(error); ok && e != nil {
			return e
		} else {
			return unknownErrorOccurred
		}
	}
	return nil
}
