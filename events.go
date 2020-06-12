package main

import (
	"time"
)

// makeEvent creates an Event from a given status code
func makeEvent(name string, status int, tags map[string]string) Event {
	e := Event{
		Name:         name,
		ServerStatus: status,
		Timestamp:    time.Now(),
		Tags:         tags,
	}

	// If our event had no (nil) tags, initialze the tags map so that
	// we don't panic if tags are added later on.
	if len(e.Tags) == 0 {
		e.Tags = make(map[string]string)
	}

	return e

}
