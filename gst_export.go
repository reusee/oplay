package main

/*
#include <gst/gst.h>
*/
import "C"

import (
	"log"
	"reflect"
	"runtime"
	"unsafe"
)

//export busCallback
func busCallback(bus *C.GstBus, msg *C.GstMessage, data C.gpointer) C.gboolean {
	if IsMessage(msg) {
		messageChan := *((*chan *C.GstMessage)(unsafe.Pointer(data)))
		copy := C.gst_message_copy(msg)
		runtime.SetFinalizer(copy, func(m *C.GstMessage) {
			C.gst_message_unref(m)
		})
		messageChan <- copy
	}
	return C.gboolean(1)
}

//export closureMarshal
func closureMarshal(closure *C.GClosure, ret *C.GValue, nParams C.guint, params *C.GValue, hint, data C.gpointer) {
	// callback value
	f := *((*interface{})(unsafe.Pointer(data)))
	fValue := reflect.ValueOf(f)
	fType := fValue.Type()
	if int(nParams) != fType.NumIn() {
		log.Fatal("number of parameters and arguments mismatch")
	}

	// convert GValue to reflect.Value
	var paramSlice []C.GValue
	h := (*reflect.SliceHeader)(unsafe.Pointer(&paramSlice))
	h.Len = int(nParams)
	h.Cap = h.Len
	h.Data = uintptr(unsafe.Pointer(params))
	var arguments []reflect.Value
	for i, gv := range paramSlice {
		goValue := fromGValue(&gv)
		var arg reflect.Value
		switch fType.In(i).Kind() {
		case reflect.Ptr:
			p := goValue.(unsafe.Pointer)
			arg = reflect.NewAt(fType.In(i), unsafe.Pointer(&p)).Elem()
		default:
			panic("FIXME") //TODO
		}
		arguments = append(arguments, arg)
	}

	// call
	fValue.Call(arguments[:fType.NumIn()])

	//TODO set return value
}

//export tagForeachCb
func tagForeachCb(list *C.GstTagList, tag *C.gchar, data C.gpointer) {
	f := *((*func(*C.gchar))(unsafe.Pointer(data)))
	f(tag)
}

//export padProbeCb
func padProbeCb(pad *C.GstPad, info *C.GstPadProbeInfo, data C.gpointer) C.GstPadProbeReturn {
	f := *((*func(*C.GstPadProbeInfo)C.GstPadProbeReturn)(unsafe.Pointer(data)))
	return f(info)
}
