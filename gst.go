package main

/*
#include <gst/gst.h>
#include <stdlib.h>
#cgo pkg-config: gstreamer-1.0

gboolean T = TRUE;
gboolean F = FALSE;

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
	return (GValue*)g_slice_alloc0(sizeof(GValue));
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

extern GstPadProbeReturn padProbeCb(GstPad*, GstPadProbeInfo*, gpointer);
void pad_add_probe(GstPad *pad, GstPadProbeType mask, void *data) {
	gst_pad_add_probe(pad, mask, padProbeCb, data, NULL);
}

*/
import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

func init() {
	C.gst_init(nil, nil)
}

// boolean

func True() C.gboolean {
	return C.T
}

func False() C.gboolean {
	return C.F
}

// Element

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

func NewElementFromUri(t C.GstURIType, uri, name string) (*C.GstElement, error) {
	var err *C.GError
	element := C.gst_element_make_from_uri(t, toGStr(uri), toGStr(name), &err)
	if element == nil {
		defer C.g_error_free(err)
		return nil, errors.New(fmt.Sprintf("%s", err.message))
	}
	runtime.SetFinalizer(element, func(e *C.GstElement) {
		C.gst_object_unref(asGPtr(element))
	})
	return element, nil
}

func ElementLink(elements ...interface{}) error {
	for i := 0; i < len(elements)-1; i++ {
		if C.gst_element_link(asGstElem(elements[i]), asGstElem(elements[i+1])) != True() {
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

func pMsg(src, t string, args ...interface{}) {
	if len(args) > 0 {
		p("%-16s %-16s %s\n", src, t, fmt.Sprintf(args[0].(string), args[1:]...))
	} else {
		p("%-16s %-16s\n", src, t)
	}
}

func MessageDump(msg *C.GstMessage) {
	srcName := fromGStr(C.gst_object_get_name(msg.src))
	switch msg._type {
	case C.GST_MESSAGE_UNKNOWN: // unknown
		pMsg(srcName, "Unknown")
	case C.GST_MESSAGE_EOS: // end of stream
		pMsg(srcName, "Eos")
	case C.GST_MESSAGE_ERROR: // error
		var err *C.GError
		var debug *C.gchar
		C.gst_message_parse_error(msg, &err, &debug)
		pMsg(srcName, "Error", "%s %s", fromGStr(err.message), fromGStr(debug))
		C.g_error_free(err)
		C.g_free(asGPtr(debug))
	//case C.GST_MESSAGE_WARNING: // warning
	//case C.GST_MESSAGE_INFO: // info
	case C.GST_MESSAGE_TAG: // tag
		var tagList *C.GstTagList
		C.gst_message_parse_tag(msg, &tagList)
		pMsg(srcName, "Tag")
		TagForeach(tagList, func(tag *C.gchar) {
			num := C.gst_tag_list_get_tag_size(tagList, tag)
			for i := C.guint(0); i < num; i++ {
				val := C.gst_tag_list_get_value_index(tagList, tag, i)
				pMsg(srcName, "Tag", "%s = %v", fromGStr(tag), fromGValue(val))
			}
		})
		C.gst_tag_list_unref(tagList)
	//case C.GST_MESSAGE_BUFFERING: // buffering
	case C.GST_MESSAGE_STATE_CHANGED: // state changed
		var oldState, newState C.GstState
		C.gst_message_parse_state_changed(msg, &oldState, &newState, nil)
		pMsg(srcName, "State", "%s -> %s",
			fromGStr(C.gst_element_state_get_name(oldState)),
			fromGStr(C.gst_element_state_get_name(newState)))
	//case C.GST_MESSAGE_STATE_DIRTY: // state dirty
	//case C.GST_MESSAGE_STEP_DONE: // step done
	//case C.GST_MESSAGE_CLOCK_PROVIDE: // clock provide
	//case C.GST_MESSAGE_CLOCK_LOST: // clock lost
	case C.GST_MESSAGE_NEW_CLOCK: // new clock
		var clock *C.GstClock
		C.gst_message_parse_new_clock(msg, &clock)
		pMsg(srcName, "New clock")
	//case C.GST_MESSAGE_STRUCTURE_CHANGE: // structure change
	case C.GST_MESSAGE_STREAM_STATUS: // stream status
		var t C.GstStreamStatusType
		var owner *C.GstElement
		C.gst_message_parse_stream_status(msg, &t, &owner)
		pMsg(srcName, "Stream status", "%d", t)
	//case C.GST_MESSAGE_APPLICATION: // application
	case C.GST_MESSAGE_ELEMENT: // element
		pMsg(srcName, "Element", "%v", msg)
	//case C.GST_MESSAGE_SEGMENT_START: // segment start
	//case C.GST_MESSAGE_SEGMENT_DONE: // segment done
	case C.GST_MESSAGE_DURATION_CHANGED: // duration changed
		pMsg(srcName, "Duration changed")
	case C.GST_MESSAGE_LATENCY: // latency
		pMsg(srcName, "Latency")
	//case C.GST_MESSAGE_ASYNC_START: // async start
	case C.GST_MESSAGE_ASYNC_DONE: // async done
		C.gst_message_parse_async_done(msg, nil)
		pMsg(srcName, "Async done")
	//case C.GST_MESSAGE_REQUEST_STATE: // request state
	//case C.GST_MESSAGE_STEP_START: // step start
	//case C.GST_MESSAGE_QOS: // qos
	//case C.GST_MESSAGE_PROGRESS: // progress
	//case C.GST_MESSAGE_TOC: // toc
	case C.GST_MESSAGE_RESET_TIME: // reset time
		C.gst_message_parse_reset_time(msg, nil)
		pMsg(srcName, "Reset time")
	case C.GST_MESSAGE_STREAM_START: // stream start
		pMsg(srcName, "Stream start")
	//case C.GST_MESSAGE_NEED_CONTEXT: // need context
	//case C.GST_MESSAGE_HAVE_CONTEXT: // have context
	//case C.GST_MESSAGE_ANY: // any
	default:
		name := C.gst_message_type_get_name(msg._type)
		p("message type %s\n", fromGStr(name))
		panic("fixme")
	}
	C.gst_message_unref(msg)
}

// Tag

func TagForeach(list *C.GstTagList, f func(*C.gchar)) {
	C.tag_foreach(list, unsafe.Pointer(&f))
}

// Caps

func NewCapsSimple(mediaType string, args ...interface{}) *C.GstCaps {
	caps := C.gst_caps_new_empty_simple(C.CString(mediaType))
	for i := 0; i < len(args); i += 2 {
		name := C.CString(args[i].(string))
		value := toGValue(args[i+1])
		C.gst_caps_set_value(caps, name, value)
	}
	return caps
}

// Pad

func PadAddProbe(pad *C.GstPad, mask C.GstPadProbeType, cb func(*C.GstPadProbeInfo) C.GstPadProbeReturn) {
	refHolderLock.Lock()
	refHolder = append(refHolder, &cb)
	refHolderLock.Unlock()
	C.pad_add_probe(pad, mask, unsafe.Pointer(&cb))
}

// Object

func ObjSet(obj *C.GObject, name string, value interface{}) {
	C.g_object_set_property(obj, toGStr(name), toGValue(value))
}

var refHolder []interface{}
var refHolderLock sync.Mutex

func ObjConnect(obj *C.GObject, signal string, cb interface{}) C.gulong {
	cbp := &cb
	refHolderLock.Lock()
	refHolder = append(refHolder, cbp) //TODO deref
	refHolderLock.Unlock()
	closure := C.new_closure(unsafe.Pointer(cbp))
	cSignal := (*C.gchar)(unsafe.Pointer(C.CString(signal)))
	defer C.free(unsafe.Pointer(cSignal))
	id := C.g_signal_connect_closure(asGPtr(obj), cSignal, closure, False())
	return id
}

// GValue

type Fraction struct {
	N int
	D int
}

var (
	gstCapsType = reflect.TypeOf(new(C.GstCaps))
)

func toGValue(v interface{}) *C.GValue {
	value := C.gvalue_new()
	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		C.g_value_init(value, C.G_TYPE_STRING)
		cStr := C.CString(v.(string))
		defer C.free(unsafe.Pointer(cStr))
		C.g_value_set_string(value, (*C.gchar)(unsafe.Pointer(cStr)))
	case reflect.Int:
		C.g_value_init(value, C.G_TYPE_INT)
		C.g_value_set_int(value, C.gint(v.(int)))
	case reflect.Struct:
		switch rv := v.(type) {
		case Fraction:
			C.g_value_init(value, C.gst_fraction_get_type())
			C.gst_value_set_fraction(value, C.gint(rv.N), C.gint(rv.D))
		default:
			p("unknown struct type %v\n", v)
			panic("fixme") //TODO
		}
	case reflect.Ptr:
		switch reflect.TypeOf(v) {
		case gstCapsType:
			C.g_value_init(value, C.gst_caps_get_type())
			C.gst_value_set_caps(value, v.(*C.GstCaps))
		default:
			panic(fmt.Sprintf("unknown type %v", v)) //TODO
		}
	default:
		panic(fmt.Sprintf("unknown type %v", v)) //TODO
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
	case C.G_TYPE_BOXED:
		ret = unsafe.Pointer(C.g_value_get_boxed(v))
	case C.G_TYPE_BOOLEAN:
		ret = C.g_value_get_boolean(v) == True()
	default:
		p("from type %s %T\n", fromGStr(C.g_type_name(fundamentalType)), v)
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

func asGstCaps(i interface{}) *C.GstCaps {
	return (*C.GstCaps)(unsafe.Pointer(reflect.ValueOf(i).Pointer()))
}

func asGstMiniObject(i interface{}) *C.GstMiniObject {
	return (*C.GstMiniObject)(unsafe.Pointer(reflect.ValueOf(i).Pointer()))
}

func asGstBuffer(i interface{}) *C.GstBuffer {
	return (*C.GstBuffer)(unsafe.Pointer(reflect.ValueOf(i).Pointer()))
}
