package flathttp

import (
	"github.com/valyala/fasttemplate"
	"reflect"
	"unsafe"
)

type F map[string]interface{}

func T(tmpl string, fields F) []byte {
	return B(fasttemplate.ExecuteString(tmpl, "[[", "]]", fields))
}

func B(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(&bh))
}
