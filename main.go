package main

/*
#include <gst/gst.h>
*/
import "C"
import (
	"log"
	"os"
	"time"
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
		sink := C.gst_element_get_static_pad(decoder, toGStr("sink"))
		C.gst_pad_link(pad, sink)
		C.gst_object_unref(asGPtr(sink))
	})
	C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING)

	go func() {
		for msg := range messages {
			srcName := fromGStr(C.gst_object_get_name(msg.src))
			switch msg._type {
			case C.GST_MESSAGE_ERROR: // error
				var err *C.GError
				var debug *C.gchar
				C.gst_message_parse_error(msg, &err, &debug)
				p("Error of %s: %s\n%s\n", srcName, fromGStr(err.message), fromGStr(debug))
				C.g_error_free(err)
				C.g_free(asGPtr(debug))
			case C.GST_MESSAGE_STATE_CHANGED: // state changed
				var oldState, newState C.GstState
				C.gst_message_parse_state_changed(msg, &oldState, &newState, nil)
				p("State of %s: %s -> %s\n", srcName,
					fromGStr(C.gst_element_state_get_name(oldState)),
					fromGStr(C.gst_element_state_get_name(newState)))
			case C.GST_MESSAGE_STREAM_STATUS: // stream status
				var t C.GstStreamStatusType
				var owner *C.GstElement
				C.gst_message_parse_stream_status(msg, &t, &owner)
				p("Stream status of %s: %d\n", srcName,
					t)
			case C.GST_MESSAGE_STREAM_START: // stream start
				p("Stream start of %s\n", srcName)
			case C.GST_MESSAGE_TAG: // tag
				var tagList *C.GstTagList
				C.gst_message_parse_tag(msg, &tagList)
				p("Tag of %s\n", srcName)
				TagForeach(tagList, func(tag *C.gchar) {
					num := C.gst_tag_list_get_tag_size(tagList, tag)
					for i := C.guint(0); i < num; i++ {
						val := C.gst_tag_list_get_value_index(tagList, tag, i)
						p("%s = %v\n", fromGStr(tag), fromGValue(val))
					}
				})
				C.gst_tag_list_unref(tagList)
			case C.GST_MESSAGE_ASYNC_DONE: // async done
				C.gst_message_parse_async_done(msg, nil)
				p("Async done of %s\n", srcName)
			case C.GST_MESSAGE_NEW_CLOCK: // new clock
				var clock *C.GstClock
				C.gst_message_parse_new_clock(msg, &clock)
				p("New clock of %s\n", srcName)
			case C.GST_MESSAGE_RESET_TIME: // reset time
				C.gst_message_parse_reset_time(msg, nil)
				p("Reset time of %s\n", srcName)
			default:
				name := C.gst_message_type_get_name(msg._type)
				p("message type %s\n", fromGStr(name))
				panic("fixme")
			}
			C.gst_message_unref(msg)
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

	loop := C.g_main_loop_new(nil, C.gboolean(0))
	C.g_main_loop_run(loop)
}
