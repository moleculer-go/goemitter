// An EventEmitter package for go 1.2 +
// Author: Mohammed Al Ashaal
// License: MIT License
// Version: v1.0.0
package Emitter

import (
	"reflect"
	"strings"
	"sync"
)

// wildcard helper
func eventMatchPattern(eventName, pattern []rune) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			return eventMatchPattern(eventName, pattern[1:]) || (len(eventName) > 0 && eventMatchPattern(eventName[1:], pattern))

		default:
			if len(eventName) == 0 || eventName[0] != pattern[0] {
				return false
			}
		}

		eventName = eventName[1:]
		pattern = pattern[1:]
	}

	return len(eventName) == 0 && len(pattern) == 0
}

// Emitter - our listeners container
type Emitter struct {
	listeners map[interface{}][]Listener
	mutex     *sync.Mutex
}

// Listener - our callback container and whether it will run once or not
type Listener struct {
	callback func(...interface{})
	once     bool
}

// Construct() - create a new instance of Emitter
func Construct() *Emitter {
	return &Emitter{
		make(map[interface{}][]Listener),
		&sync.Mutex{},
	}
}

// Destruct() - free memory from an emitter instance
func (self *Emitter) Destruct() {
	self = nil
}

// AddListener() - register a new listener on the specified event
func (self *Emitter) AddListener(event string, callback func(...interface{})) *Emitter {
	return self.On(event, callback)
}

// On() - register a new listener on the specified event
func (self *Emitter) On(event string, callback func(...interface{})) *Emitter {
	self.mutex.Lock()
	if _, ok := self.listeners[event]; !ok {
		self.listeners[event] = []Listener{}
	}
	self.listeners[event] = append(self.listeners[event], Listener{callback, false})
	self.mutex.Unlock()

	self.EmitSync("newListener", []interface{}{event, callback})
	return self
}

// Once() - register a new one-time listener on the specified event
func (self *Emitter) Once(event string, callback func(...interface{})) *Emitter {
	self.mutex.Lock()
	if _, ok := self.listeners[event]; !ok {
		self.listeners[event] = []Listener{}
	}
	self.listeners[event] = append(self.listeners[event], Listener{callback, true})
	self.mutex.Unlock()

	self.EmitSync("newListener", []interface{}{event, callback})
	return self
}

// RemoveListeners() - remove the specified callback from the specified events' listeners
func (self *Emitter) RemoveListener(event string, callback func(...interface{})) *Emitter {
	return self.removeListenerInternal(event, callback, false)
}

func (self *Emitter) removeListenerInternal(event string, callback func(...interface{}), suppress bool) *Emitter {
	self.mutex.Lock()

	if _, ok := self.listeners[event]; !ok {
		self.mutex.Unlock()
		return self
	}

	for k, v := range self.listeners[event] {
		if reflect.ValueOf(v.callback).Pointer() == reflect.ValueOf(callback).Pointer() {
			self.listeners[event] = append(self.listeners[event][:k], self.listeners[event][k+1:]...)

			self.mutex.Unlock()

			if !suppress {
				self.EmitSync("removeListener", []interface{}{event, callback})
			}
			return self
		}
	}

	self.mutex.Unlock()

	return self
}

// RemoveAllListeners() - remove all listeners from (all/event)
func (self *Emitter) RemoveAllListeners(event interface{}) *Emitter {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	if event == nil {
		self.listeners = make(map[interface{}][]Listener)
		return self
	}
	if _, ok := self.listeners[event]; !ok {
		return self
	}
	delete(self.listeners, event)
	return self
}

// Listeners() - return an array with the registered listeners in the specified event
func (self *Emitter) Listeners(event string) []Listener {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	listeners := make([]Listener, 0)

	// add the ones that follow pattern
	for eventPattern, lis := range self.listeners {
		shouldAdd := false

		// add generic "**" bound listeners
		shouldAdd = shouldAdd || eventPattern.(string) == "**"
		// add listener bound on full name event
		shouldAdd = shouldAdd || eventPattern.(string) == event
		// add listeners that have matching wildcard pattern
		shouldAdd = shouldAdd ||
			(strings.Contains(eventPattern.(string), "*") &&
				eventMatchPattern([]rune(event), []rune(eventPattern.(string))))

		if shouldAdd {
			listeners = append(listeners, lis...)
		}
	}

	return listeners
}

// ListenersCount() - return the count of listeners in the speicifed event
func (self *Emitter) ListenersCount(event string) int {
	return len(self.Listeners(event))
}

// EmitSync() - run all listeners of the specified event in synchronous mode
func (self *Emitter) EmitSync(event string, args ...interface{}) *Emitter {
	for _, v := range self.Listeners(event) {
		if v.once {
			self.removeListenerInternal(event, v.callback, true)
		}
		v.callback(args...)
	}

	return self
}

// EmitAsync() - run all listeners of the specified event in asynchronous mode using goroutines
func (self *Emitter) EmitAsync(event string, args []interface{}) *Emitter {
	for _, v := range self.Listeners(event) {
		if v.once {
			self.removeListenerInternal(event, v.callback, true)
		}
		go v.callback(args...)
	}
	return self
}
