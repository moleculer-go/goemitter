package Emitter

import (
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRemoveListener(t *testing.T) {
	emitter := Construct()

	counter := 0
	fn1 := func(args ...interface{}) {
		counter++
	}
	fn2 := func(args ...interface{}) {
		counter++
	}

	emitter.On("testevent", fn1)
	emitter.On("testevent", fn2)

	emitter.RemoveListener("testevent", fn1)
	emitter.EmitSync("testevent")

	listenersCount := emitter.ListenersCount("testevent")

	expect(t, 1, listenersCount)
	expect(t, 1, counter)
}

func TestOnce(t *testing.T) {
	emitter := Construct()

	counter := 0
	fn := func(args ...interface{}) {
		counter++
	}

	emitter.Once("testevent", fn)

	emitter.EmitSync("testevent")
	emitter.EmitSync("testevent")

	expect(t, 1, counter)
}

func TestRandomConcurrentCalls(t *testing.T) {
	emitter := Construct()

	var counter int32
	var err error

	randomCallsFn := func() {
		defer func() {
			if r := recover(); r != nil {
				err = r.(error)
			}
		}()

		fn1 := func(args ...interface{}) {
			atomic.AddInt32(&counter, 1)
		}
		fn2 := func(args ...interface{}) {
			atomic.AddInt32(&counter, 1)
		}

		events := []string{"event1", "event2", "event3"}
		fns := []func(...interface{}){fn1, fn2}

		m := map[int]interface{}{}
		for i := 0; i < 100; i++ {
			eventIdx := int(rand.Int31()) % len(events)
			fnIdx := int(rand.Int31()) % len(fns)
			key := fnIdx<<4 + eventIdx

			action := int(rand.Int31())
			if action%3 == 0 {
				if _, ok := m[key]; !ok {
					emitter.On(events[eventIdx], fns[fnIdx])
					m[key] = nil
				}
			} else if action%7 == 0 {
				emitter.RemoveListener(events[eventIdx], fns[fnIdx])
				delete(m, key)
			} else {
				emitter.EmitAsync(events[eventIdx], nil)
			}
		}
	}

	wg := sync.WaitGroup{}
	for j := 0; j < 10; j++ {
		go func() {
			wg.Add(1)
			randomCallsFn()
			wg.Done()
		}()
	}
	wg.Wait()

	expect(t, nil, err)
}

func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}
