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

var valueBufPool sync.Pool

func acquireValueBuffer(size int) []interface{} {
	val := valueBufPool.Get()
	if val == nil {
		val = make([]interface{}, 0, 0)
	}

	buf := val.([]interface{})

	if n := size - cap(buf); n > 0 {
		buf = append(buf[:cap(buf)], make([]interface{}, n)...)

		for i := len(buf) - n; i < len(buf); i++ {
			var val interface{}
			buf[i] = &val
		}
	}

	return buf[:size]
}

func releaseValueBuffer(buf []interface{}) {
	valueBufPool.Put(buf)
}
