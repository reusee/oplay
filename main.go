package main

/*
#include <gst/gst.h>
*/
import "C"
import (
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unsafe"
)

func main() {
	playbin()
	//ranged()
	//probe()
	//oggplayer()
}

// playbin
func playbin() {
	pipeline, err := NewElement("playbin", "playbin")
	if err != nil {
		log.Fatal(err)
	}
	uri := os.Args[1]
	if !strings.Contains(uri, "://") {
		uri, err = filepath.Abs(uri)
		if err != nil {
			log.Fatal(err)
		}
		uri = "file://" + uri
	}
	ObjSet(asGObj(pipeline), "uri", uri)

	messages := PipelineWatchBus(asGstPipeline(pipeline))
	go func() {
		for msg := range messages {
			MessageDump(msg)
		}
	}()

	C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING)

	loop := C.g_main_loop_new(nil, False())
	C.g_main_loop_run(loop)
}

// ranged playing
func ranged() {
	pipeline := C.gst_pipeline_new(toGStr("pipeline"))
	src, err := NewElement("uridecodebin", "src")
	if err != nil {
		log.Fatal(err)
	}
	csp, err := NewElement("videoconvert", "csp")
	if err != nil {
		log.Fatal(err)
	}
	vs, err := NewElement("videoscale", "vs")
	if err != nil {
		log.Fatal(err)
	}
	sink, err := NewElement("autovideosink", "sink")
	if err != nil {
		log.Fatal(err)
	}

	ObjSet(asGObj(src), "uri", os.Args[1])
	BinAdd(pipeline, src, csp, vs, sink)
	ElementLink(csp, vs, sink)
	//TODO sinkpad :=

	ObjConnect(asGObj(src), "pad-added", func(e *C.GstElement, pad *C.GstPad) {
		p("====\n")
		PadAddProbe(pad, C.GST_PAD_PROBE_TYPE_BLOCK_DOWNSTREAM, func(info *C.GstPadProbeInfo) C.GstPadProbeReturn {
			p("blocked\n")
			return C.GST_PAD_PROBE_OK
		})
		//C.gst_pad_link(pad, sinkpad)
	})
	ObjConnect(asGObj(src), "no-more-pads", func(e *C.GstElement) {
	})

	C.gst_element_set_state(pipeline, C.GST_STATE_PAUSED)

	messages := PipelineWatchBus(asGstPipeline(pipeline))
	go func() {
		for msg := range messages {
			MessageDump(msg)
		}
	}()

	loop := C.g_main_loop_new(nil, False())
	C.g_main_loop_run(loop)
}

// pad probe
func probe() {
	pipeline := C.gst_pipeline_new(toGStr("pipeline"))
	src, err := NewElement("videotestsrc", "src")
	if err != nil {
		log.Fatal(err)
	}
	filter, err := NewElement("capsfilter", "filter")
	if err != nil {
		log.Fatal(err)
	}
	csp, err := NewElement("videoconvert", "csp")
	if err != nil {
		log.Fatal(err)
	}
	sink, err := NewElement("xvimagesink", "sink")
	if err != nil {
		log.Fatal(err)
	}

	BinAdd(pipeline, src, filter, csp, sink)
	ElementLink(src, filter, csp, sink)
	C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING)
	messages := PipelineWatchBus(asGstPipeline(pipeline))

	width, height := 640, 480
	filtercaps := NewCapsSimple("video/x-raw",
		"format", "RGB16",
		"width", width,
		"height", height,
		"framerate", Fraction{25, 1})
	ObjSet(asGObj(filter), "caps", filtercaps)
	n := C.gst_caps_get_size(filtercaps)
	structure := C.gst_caps_get_structure(filtercaps, n-1)
	_ = structure
	C.gst_caps_unref(filtercaps)

	pad := C.gst_element_get_static_pad(src, toGStr("src"))
	PadAddProbe(pad, C.GST_PAD_PROBE_TYPE_BUFFER, func(info *C.GstPadProbeInfo) C.GstPadProbeReturn {
		buffer := (*C.GstBuffer)(unsafe.Pointer(info.data))
		buffer = asGstBuffer(C.gst_mini_object_make_writable(asGstMiniObject(buffer)))
		var mapInfo C.GstMapInfo
		C.gst_buffer_map(buffer, &mapInfo, C.GST_MAP_WRITE)
		var ptr []C.guint16
		header := (*reflect.SliceHeader)(unsafe.Pointer(&ptr))
		header.Len = int(mapInfo.size / 2)
		header.Cap = header.Len
		header.Data = uintptr(unsafe.Pointer(mapInfo.data))
		for y := 0; y < height; y++ {
			for x := 0; x < width/2; x++ {
				ptr[x], ptr[width-1-x] = ptr[width-1-x], ptr[x]
			}
			ptr = ptr[width:]
		}
		C.gst_buffer_unmap(buffer, &mapInfo)
		info.data = asGPtr(buffer)
		return C.GST_PAD_PROBE_OK
	})
	C.gst_object_unref(asGPtr(pad))

	go func() {
		for msg := range messages {
			MessageDump(msg)
		}
	}()

	loop := C.g_main_loop_new(nil, False())
	C.g_main_loop_run(loop)
}

func oggplayer() {
	pipeline := C.gst_pipeline_new(toGStr("audio-player"))
	source, err := NewElement("filesrc", "file-source")
	if err != nil {
		log.Fatal(err)
	}
	demuxer, err := NewElement("oggdemux", "ogg-demuxer")
	if err != nil {
		log.Fatal(err)
	}
	decoder, err := NewElement("vorbisdec", "vorbis-decoder")
	if err != nil {
		log.Fatal(err)
	}
	conv, err := NewElement("audioconvert", "converter")
	if err != nil {
		log.Fatal(err)
	}
	sink, err := NewElement("autoaudiosink", "audio-output")
	if err != nil {
		log.Fatal(err)
	}

	ObjSet(asGObj(source), "location", os.Args[1])
	messages := PipelineWatchBus(asGstPipeline(pipeline))
	BinAdd(pipeline, source, demuxer, decoder, conv, sink)
	ElementLink(source, demuxer)
	ElementLink(decoder, conv, sink)
	ObjConnect(asGObj(demuxer), "pad-added", func(elem *C.GstElement, pad *C.GstPad) {
		sink := C.gst_element_get_static_pad(decoder, toGStr("sink"))
		C.gst_pad_link(pad, sink)
		C.gst_object_unref(asGPtr(sink))
	})
	C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING)

	go func() {
		for msg := range messages {
			MessageDump(msg)
		}
	}()

	go func() {
		var pos, length C.gint64
		for _ = range time.NewTicker(time.Second * 1).C {
			C.gst_element_query_position(pipeline, C.GST_FORMAT_TIME, &pos)
			C.gst_element_query_duration(pipeline, C.GST_FORMAT_TIME, &length)
			p("%v / %v\n", time.Duration(pos), time.Duration(length))
			if time.Duration(pos) > time.Second*5 {
				C.gst_element_seek(pipeline, 1.0, C.GST_FORMAT_TIME, C.GST_SEEK_FLAG_FLUSH,
					C.GST_SEEK_TYPE_SET, 10,
					C.GST_SEEK_TYPE_NONE, C.GST_CLOCK_TIME_NONE)
			}
		}
	}()

	loop := C.g_main_loop_new(nil, False())
	C.g_main_loop_run(loop)
}
