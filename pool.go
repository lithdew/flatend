package flatend

import (
	"sync"
)

type Values map[string]interface{}

var valuesPool sync.Pool

func acquireValues() Values {
	v := valuesPool.Get()
	if v == nil {
		v = make(Values)
	}
	return v.(Values)
}

func releaseValues(values Values) {
	for k := range values {
		delete(values, k)
	}
	valuesPool.Put(values)
}
