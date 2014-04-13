package main

/*
#include <gst/gst.h>
*/
import "C"

import (
	"unsafe"
)

//export busCallback
func busCallback(bus *C.GstBus, msg *C.GstMessage, data C.gpointer) C.gboolean {
	messageChan := *((*chan *C.GstMessage)(unsafe.Pointer(data)))
	messageChan <- msg
	return C.gboolean(1)
}
