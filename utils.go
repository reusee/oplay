package main

import "log"

func init() {
	log.SetFlags(log.Ltime)
	log.SetPrefix("*")
}

func p(fmt string, args ...interface{}) {
	log.Printf(fmt, args...)
}
