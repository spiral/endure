package endure

import (
	"context"
	"reflect"

	"github.com/spiral/endure/pkg/fsm"
	"github.com/spiral/endure/pkg/linked_list"
	"github.com/spiral/endure/pkg/vertex"
	"github.com/spiral/errors"
	"go.uber.org/zap"
)

func (e *Endure) internalStop(vID string) error {
	const op = errors.Op("endure_internal_stop")
	vrtx := e.graph.GetVertex(vID)
	if reflect.TypeOf(vrtx.Iface).Implements(reflect.TypeOf((*Service)(nil)).Elem()) {
		in := make([]reflect.Value, 0, 1)
		// add service itself
		in = append(in, reflect.ValueOf(vrtx.Iface))

		err := e.callStopFn(vrtx, in)
		if err != nil {
			e.logger.Error("error occurred during the callStopFn", zap.String("vertex id", vrtx.ID))
			return errors.E(op, errors.FunctionCall, err)
		}
	}
	return nil
}

func (e *Endure) callStopFn(vrtx *vertex.Vertex, in []reflect.Value) error {
	const op = errors.Op("endure_call_stop_fn")
	// Call Stop() method, which returns only error (or nil)
	e.logger.Debug("calling internal_stop function on the vrtx", zap.String("vrtx id", vrtx.ID))
	m, _ := reflect.TypeOf(vrtx.Iface).MethodByName(StopMethodName)
	ret := m.Func.Call(in)
	rErr := ret[0].Interface()
	if rErr != nil {
		if err, ok := rErr.(error); ok && err != nil {
			return errors.E(op, errors.FunctionCall, e)
		}
		return errors.E(op, errors.FunctionCall, errors.Str("unknown error occurred during the function call"))
	}
	return nil
}

// true -> next
// false -> prev
func (e *Endure) shutdown(n *linked_list.DllNode, traverseNext bool) error {
	const op = errors.Op("endure_shutdown")
	numOfVertices := calculateDepth(n, traverseNext)
	if numOfVertices == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.stopTimeout)
	defer cancel()
	c := make(chan string)

	// used to properly exit
	// if the total number of vertices equal to the stopped, it means, that we stopped all
	stopped := 0

	go func() {
		// process all nodes one by one
		nCopy := n
		for nCopy != nil {
			go func(v *vertex.Vertex) {
				// if vertex is disabled, just skip it, but send to the channel ID
				if v.IsDisabled == true {
					c <- v.ID
					return
				}

				// if vertex is Uninitialized or already stopped
				// Skip vertices which are not Started
				if v.GetState() != fsm.Started {
					c <- v.ID
					return
				}

				v.SetState(fsm.Stopping)

				// if we have a running poller, exit from it
				tmp, ok := e.results.Load(v.ID)
				if ok {
					channel := tmp.(*result)

					// exit from vertex poller
					channel.signal <- notify{}
					e.results.Delete(v.ID)
				}

				// call Stop on the Vertex
				err := e.internalStop(v.ID)
				if err != nil {
					v.SetState(fsm.Error)
					c <- v.ID
					e.logger.Error("error stopping vertex", zap.String("vertex id", v.ID), zap.Error(err))
					return
				}
				v.SetState(fsm.Stopped)
				c <- v.ID
			}(nCopy.Vertex)
			if traverseNext {
				nCopy = nCopy.Next
			} else {
				nCopy = nCopy.Prev
			}
		}
	}()

	for {
		select {
		// get notification about stopped vertex
		case vid := <-c:
			e.logger.Info("vertex stopped", zap.String("vertex id", vid))
			stopped++
			if stopped == numOfVertices {
				return nil
			}
		case <-ctx.Done():
			e.logger.Info("timeout exceed, some vertices are not stopped", zap.Error(ctx.Err()))
			// iterate to see vertices, which are not stopped
			VIDs := make([]string, 0, 1)
			for i := 0; i < len(e.graph.Vertices); i++ {
				state := e.graph.Vertices[i].GetState()
				if state == fsm.Started || state == fsm.Stopping {
					VIDs = append(VIDs, e.graph.Vertices[i].ID)
				}
			}
			if len(VIDs) > 0 {
				e.logger.Error("vertices which are not stopped", zap.Any("vertex id", VIDs))
			}

			return errors.E(op, errors.TimeOut, errors.Str("timeout exceed, some vertices may not be stopped and can cause memory leak"))
		}
	}
}

// Using to calculate number of Vertices in DLL
func calculateDepth(n *linked_list.DllNode, traverse bool) int {
	num := 0
	if traverse {
		tmp := n
		for tmp != nil {
			num++
			tmp = tmp.Next
		}
		return num
	}
	tmp := n
	for tmp != nil {
		num++
		tmp = tmp.Prev
	}
	return num
}
