package main

/*
#include <gst/gst.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

func NewElement(factory string, name string) (*C.GstElement, error) {
	cFactory := toGStr(factory)
	cName := toGStr(name)
	element := C.gst_element_factory_make(cFactory, cName)
	if element == nil {
		return nil, errors.New(fmt.Sprintf("failed to create element %s:%s", factory, name))
	}
	runtime.SetFinalizer(element, func(e *C.GstElement) {
		C.gst_object_unref(asGPtr(element))
	})
	return element, nil
}

func NewFactory(name string) (*C.GstElementFactory, error) {
	factory := C.gst_element_factory_find(toGStr(name))
	if factory == nil {
		return nil, errors.New(fmt.Sprintf("failed to find factory %s", name))
	}
	return factory, nil
}

func AddToBin(bin interface{}, elements ...interface{}) {
	cBin := asGstBin(bin)
	for _, e := range elements {
		C.gst_bin_add(cBin, asGstElem(e))
	}
}

func ElementLink(elements ...interface{}) error {
	for i := 0; i < len(elements)-1; i++ {
		if C.gst_element_link(asGstElem(elements[i]), asGstElem(elements[i+1])) != C.gboolean(1) {
			return errors.New("link error")
		}
	}
	return nil
}

func toGStr(s string) *C.gchar {
	return (*C.gchar)(unsafe.Pointer(C.CString(s)))
}

func fromGStr(s *C.gchar) string {
	return C.GoString((*C.char)(unsafe.Pointer(s)))
}

func asGStr(s interface{}) *C.gchar {
	return (*C.gchar)(unsafe.Pointer(reflect.ValueOf(s).Pointer()))
}

func asGPtr(i interface{}) C.gpointer {
	return (C.gpointer)(unsafe.Pointer(reflect.ValueOf(i).Pointer()))
}

func asGstObj(i interface{}) *C.GstObject {
	return (*C.GstObject)(unsafe.Pointer(reflect.ValueOf(i).Pointer()))
}

func asGstElem(i interface{}) *C.GstElement {
	return (*C.GstElement)(unsafe.Pointer(reflect.ValueOf(i).Pointer()))
}

func asGstBin(i interface{}) *C.GstBin {
	return (*C.GstBin)(unsafe.Pointer(reflect.ValueOf(i).Pointer()))
}

func asGstPipeline(i interface{}) *C.GstPipeline {
	return (*C.GstPipeline)(unsafe.Pointer(reflect.ValueOf(i).Pointer()))
}
