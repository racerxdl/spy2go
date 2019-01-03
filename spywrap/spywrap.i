/* spywrap.i */
%module spywrap
%include "typemaps.i"
%include "stdint.i"

%go_import("github.com/mattn/go-pointer")

%typemap(gotype) uint32_t *INOUT "[]uint32"

%apply uint8_t *OUTPUT { uint8_t *value };
%apply char *INOUT { char* version };
%apply uint32_t* INOUT { uint32_t* buffer };

%{
#include "airspy/airspy.h"
#include "airspy/wrap.h"
%}

%insert(cgo_comment_typedefs) %{
#cgo LDFLAGS: -l:libairspy.a -lusb-1.0
%}

%include "./airspy/airspy.h"
%include "./airspy/wrap.h"

%insert(go_wrapper) %{

type Callback struct {
	Func func(interface{}, Airspy_transfer_t) int
	Data interface{}
}

//export cbProxy
func cbProxy(v unsafe.Pointer, d SwigcptrAirspy_transfer_t) int {
	cb := pointer.Restore(v).(*Callback)
	return cb.Func(cb.Data, d)
}

%}
