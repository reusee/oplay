package main

/*
#include <gst/gst.h>
#include <stdlib.h>
#cgo pkg-config: gstreamer-1.0

extern void closureMarshal(GClosure*, GValue*, guint, GValue*, gpointer, gpointer);

GClosure* new_closure(void *data) {
	GClosure *closure = g_closure_new_simple(sizeof(GClosure), NULL);
	g_closure_set_meta_marshal(closure, data, (GClosureMarshal)(closureMarshal));
	return closure;
}

static inline GType gvalue_get_type(GValue *v) {
	return G_VALUE_TYPE(v);
}

static inline GType gtype_get_fundamental(GType t) {
	return G_TYPE_FUNDAMENTAL(t);
}

static inline GValue* gvalue_new() {
	return (GValue*)malloc(sizeof(GValue));
}

static inline const gchar* gvalue_get_type_name(GValue *v) {
	return G_VALUE_TYPE_NAME(v);
}

static inline int is_message(GstMessage *msg) {
	return GST_IS_MESSAGE(msg);
}

extern gboolean busCallback(GstBus*, GstMessage*, gpointer);
guint add_bus_watch(GstBus *bus, void *data) {
	return gst_bus_add_watch(bus, busCallback, data);
}

extern void tagForeachCb(const GstTagList*, const gchar*, gpointer);
void tag_foreach(GstTagList *list, void *data) {
	gst_tag_list_foreach(list, tagForeachCb, data);
}

*/
import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

func init() {
	C.gst_init(nil, nil)
}

// Element

func NewElement(factory string, name string) (*C.GstElement, error) {
	cFactory := toGStr(factory)
	cName := toGStr(name)
	element := C.gst_element_factory_make(cFactory, cName)
	if element == nil {
		return nil, errors.New(fmt.Sprintf("failed to create element %s:%s", factory, name))
	}
	/*
		runtime.SetFinalizer(element, func(e *C.GstElement) {
			C.gst_object_unref(asGPtr(element))
		})
	*/ //TODO
	return element, nil
}

func ElementLink(elements ...interface{}) error {
	for i := 0; i < len(elements)-1; i++ {
		if C.gst_element_link(asGstElem(elements[i]), asGstElem(elements[i+1])) != C.gboolean(1) {
			return errors.New("link error")
		}
	}
	return nil
}

// Factory

func NewFactory(name string) (*C.GstElementFactory, error) {
	factory := C.gst_element_factory_find(toGStr(name))
	if factory == nil {
		return nil, errors.New(fmt.Sprintf("failed to find factory %s", name))
	}
	return factory, nil
}

// Bin

func BinAdd(bin interface{}, elements ...interface{}) {
	cBin := asGstBin(bin)
	for _, e := range elements {
		C.gst_bin_add(cBin, asGstElem(e))
	}
}

// Pipeline

func PipelineWatchBus(pipeline *C.GstPipeline) chan *C.GstMessage {
	bus := C.gst_pipeline_get_bus(pipeline)
	defer C.gst_object_unref(asGPtr(bus))
	messages := make(chan *C.GstMessage)
	C.add_bus_watch(bus, unsafe.Pointer(&messages))
	return messages
}

// Message

func IsMessage(msg *C.GstMessage) bool {
	return C.is_message(msg) == 1
}

// Tag

func TagForeach(list *C.GstTagList, f func(*C.gchar)) {
	C.tag_foreach(list, unsafe.Pointer(&f))
}

// Object

func ObjSet(obj *C.GObject, name string, value interface{}) {
	C.g_object_set_property(obj, toGStr(name), toGValue(value))
}

var cbHolder []*interface{}
var cbLocker sync.Mutex

func ObjConnect(obj *C.GObject, signal string, cb interface{}) C.gulong {
	cbp := &cb
	cbLocker.Lock()
	cbHolder = append(cbHolder, cbp) //TODO deref
	cbLocker.Unlock()
	closure := C.new_closure(unsafe.Pointer(cbp))
	cSignal := (*C.gchar)(unsafe.Pointer(C.CString(signal)))
	defer C.free(unsafe.Pointer(cSignal))
	id := C.g_signal_connect_closure(asGPtr(obj), cSignal, closure, C.gboolean(0))
	return id
}

// GValue

func toGValue(v interface{}) *C.GValue {
	value := C.gvalue_new()
	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		C.g_value_init(value, C.G_TYPE_STRING)
		cStr := C.CString(v.(string))
		defer C.free(unsafe.Pointer(cStr))
		C.g_value_set_string(value, (*C.gchar)(unsafe.Pointer(cStr)))
	default:
		panic(fmt.Sprintf("unknown type %v", reflect.TypeOf(v).Kind()))
		//TODO more types
	}
	return value
}

func fromGValue(v *C.GValue) (ret interface{}) {
	valueType := C.gvalue_get_type(v)
	fundamentalType := C.gtype_get_fundamental(valueType)
	switch fundamentalType {
	case C.G_TYPE_OBJECT:
		ret = unsafe.Pointer(C.g_value_get_object(v))
	case C.G_TYPE_STRING:
		ret = fromGStr(C.g_value_get_string(v))
	case C.G_TYPE_UINT:
		ret = int(C.g_value_get_uint(v))
	default:
		p("from type %s\n", fromGStr(C.g_type_name(fundamentalType)))
		panic("FIXME") //TODO
	}
	return
}

func ValueGetType(v *C.GValue) C.GType {
	return C.gvalue_get_type(v)
}

func ValueGetTypeName(v *C.GValue) string {
	return fromGStr(C.gvalue_get_type_name(v))
}

// conversion

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

func asGObj(i interface{}) *C.GObject {
	return (*C.GObject)(unsafe.Pointer(reflect.ValueOf(i).Pointer()))
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
