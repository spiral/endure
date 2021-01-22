package endure

import (
	"reflect"

	"github.com/spiral/errors"
	"go.uber.org/zap"
)

func (e *Endure) fnCallCollectors(vertex *Vertex, in []reflect.Value, methodName string) error {
	const op = errors.Op("internal_call_collector_functions")
	// type implements Collector interface
	if reflect.TypeOf(vertex.Iface).Implements(reflect.TypeOf((*Collector)(nil)).Elem()) {
		// if type implements Collector() it should has FnsProviderToInvoke
		m, ok := reflect.TypeOf(vertex.Iface).MethodByName(methodName)
		if !ok {
			e.logger.Error("type has missing method in CollectorEntries", zap.String("vertex id", vertex.ID), zap.String("method", methodName))
			return errors.E(op, errors.FunctionCall, errors.Str("type has missing method in CollectorEntries"))
		}

		ret := m.Func.Call(in)
		for i := 0; i < len(ret); i++ {
			// try to find possible errors
			r := ret[i].Interface()
			if r == nil {
				continue
			}
			if rErr, ok := r.(error); ok {
				if rErr != nil {
					if err, ok := rErr.(error); ok && e != nil {
						e.logger.Error("error calling CollectorFns", zap.String("vertex id", vertex.ID), zap.Error(err))
						return errors.E(op, errors.FunctionCall, err)
					}
					return errors.E(op, errors.FunctionCall, errors.Str("unknown error occurred during the function call"))
				}
			}
		}
	}
	return nil
}