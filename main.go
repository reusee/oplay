package main

/*
#include <gst/gst.h>
*/
import "C"
import (
	"log"
	"os"
)

func main() {
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
		name := C.gst_object_get_name(asGstObj(pad))
		p("%s\n", fromGStr(name))
		sink := C.gst_element_get_static_pad(decoder, toGStr("sink"))
		C.gst_pad_link(pad, sink)
		C.gst_object_unref(asGPtr(sink))
	})
	C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING)
	go func() {
		for msg := range messages {
			switch msg._type {
			case C.GST_MESSAGE_ERROR:
				var err *C.GError
				var debug *C.gchar
				C.gst_message_parse_error(msg, &err, &debug)
				p("Error: %s\n%s\n", fromGStr(err.message), fromGStr(debug))
				C.g_error_free(err)
				C.g_free(asGPtr(debug))
			}
		}
	}()

	loop := C.g_main_loop_new(nil, C.gboolean(0))
	C.g_main_loop_run(loop)
}
