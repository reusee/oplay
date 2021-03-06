package main

/*
#include <gst/gst.h>
*/
import "C"
import (
	"log"
	"os"
	"runtime"
)

func runPlay() {
	play()
	//play2()
}

func play2() {
	pipeline := C.gst_pipeline_new(toGStr("pipeline"))

	source, err := NewElement("filesrc", "source")
	if err != nil {
		log.Fatal(err)
	}
	ObjSet(asGObj(source), "location", os.Args[1])

	demux, err := NewElement("oggdemux", "demuxer")
	if err != nil {
		log.Fatal(err)
	}

	BinAdd(pipeline, source, demux)
	C.gst_element_link_pads(source, toGStr("src"), demux, toGStr("sink"))
	ObjConnect(asGObj(demux), "pad-added", func(demuxer *C.GstElement, pad *C.GstPad) {
		p("pad added %T %v\n", pad, pad)
	})

	// bus
	messages := PipelineWatchBus(asGstPipeline(pipeline))

	C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING)

	loop := C.g_main_loop_new(nil, 0)
	go func() {
	loop:
		for msg := range messages {
			runtime.GC()
			p("=> %s: %s\n", fromGStr(C.gst_object_get_name(asGstObj(msg.src))),
				fromGStr(C.gst_message_type_get_name(msg._type)))
			switch msg._type {

			case C.GST_MESSAGE_ERROR:
				var err *C.GError
				var debug *C.gchar
				C.gst_message_parse_error(msg, &err, &debug)
				p("Error: %s\n%s\n", fromGStr(err.message), fromGStr(debug))
				C.g_error_free(err)
				C.g_free(asGPtr(debug))
				C.g_main_loop_quit(loop)
				break loop
			case C.GST_MESSAGE_WARNING:
			case C.GST_MESSAGE_INFO:

			case C.GST_MESSAGE_EOS:
				C.g_main_loop_quit(loop)
				break loop

			case C.GST_MESSAGE_TAG:
				var tags *C.GstTagList
				C.gst_message_parse_tag(msg, &tags)
				//TagListForeach(tags, ) TODO
				C.gst_tag_list_unref(tags)

			case C.GST_MESSAGE_STATE_CHANGED:
				var newState, oldState C.GstState
				C.gst_message_parse_state_changed(msg, &oldState, &newState, nil)
				p("%s -> %s\n", fromGStr(C.gst_element_state_get_name(oldState)),
					fromGStr(C.gst_element_state_get_name(newState)))

			}

			C.gst_message_unref(msg)
		}
		p("loop break.\n")
	}()

	runtime.LockOSThread()
	C.g_main_loop_run(loop)
	p("quit.\n")
}

func play() {
	// element
	e, err := NewElement("fakesrc", "source")
	if err != nil {
		log.Fatal(err)
	}
	p("%v\n", e)

	name := C.gst_object_get_name(asGstObj(e))
	p("%s\n", fromGStr(name))

	// factory
	factory, err := NewFactory("fakesrc")
	if err != nil {
		log.Fatal(err)
	}
	p("%s\n", fromGStr(C.gst_object_get_name(asGstObj(factory))))
	p("%s\n", fromGStr(C.gst_element_factory_get_metadata(factory, asGStr(C.CString(C.GST_ELEMENT_METADATA_KLASS)))))
	p("%s\n", fromGStr(C.gst_element_factory_get_metadata(factory, asGStr(C.CString(C.GST_ELEMENT_METADATA_DESCRIPTION)))))

	// id pipeline
	pipeline := C.gst_pipeline_new(toGStr("pipeline"))
	source, _ := NewElement("fakesrc", "source")
	filter, _ := NewElement("identity", "filter")
	sink, _ := NewElement("fakesink", "sink")
	BinAdd(pipeline, source, filter, sink)
	ElementLink(source, filter, sink)
	C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING)
	C.gst_element_sync_state_with_parent(filter)

	/*
		// bin
		bin := C.gst_bin_new(toGStr("bin"))
		source, _ = NewElement("fakesrc", "source")
		sink, _ = NewElement("fakesink", "sink")
		BinAdd(bin, source, sink)
		//C.gst_bin_remove(asGstBin(bin), source)
		ElementLink(source, sink)
		BinAdd(pipeline, bin)
		source = C.gst_bin_get_by_name(asGstBin(bin), toGStr("source"))
	*/

	// bus
	messages := PipelineWatchBus(asGstPipeline(pipeline))
	loop := C.g_main_loop_new(nil, 0)
	go func() {
		C.g_main_loop_run(loop)
	}()

	// message
loop:
	for msg := range messages {
		runtime.GC()
		p("=> %s from %s\n", fromGStr(C.gst_message_type_get_name(msg._type)),
			fromGStr(C.gst_object_get_name(asGstObj(msg.src))))
		switch msg._type {

		case C.GST_MESSAGE_ERROR:
			var err *C.GError
			var debug *C.gchar
			C.gst_message_parse_error(msg, &err, &debug)
			p("Error: %s\n%s\n", fromGStr(err.message), fromGStr(debug))
			C.g_error_free(err)
			C.g_free(asGPtr(debug))
			C.g_main_loop_quit(loop)
			break loop
		case C.GST_MESSAGE_WARNING:
		case C.GST_MESSAGE_INFO:

		case C.GST_MESSAGE_EOS:
			C.g_main_loop_quit(loop)
			break loop

		case C.GST_MESSAGE_TAG:
			var tags *C.GstTagList
			C.gst_message_parse_tag(msg, &tags)
			//TagListForeach(tags, ) TODO
			C.gst_tag_list_unref(tags)

		case C.GST_MESSAGE_STATE_CHANGED:
			var newState, oldState C.GstState
			C.gst_message_parse_state_changed(msg, &oldState, &newState, nil)
			p("%s -> %s\n", fromGStr(C.gst_element_state_get_name(oldState)),
				fromGStr(C.gst_element_state_get_name(newState)))

		}
	}
}
