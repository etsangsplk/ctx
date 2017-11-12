package ctx

import "sync"

// Doner can block until something is done
type Doner interface {
	Done() <-chan struct{}
}

type doneChan <-chan struct{}

func (dc doneChan) Done() <-chan struct{} { return dc }

// Lift takes a chan and wraps it in a Doner
func Lift(c <-chan struct{}) Doner { return doneChan(c) }

// Tick returns a <-chan whose range ends when the underlying context cancels
func Tick(d Doner) <-chan struct{} {
	cq := make(chan struct{})
	go func() {
		for {
			select {
			case <-d.Done():
				close(cq)
				return
			case cq <- struct{}{}:
			}
		}
	}()
	return cq
}

// Defer guarantees that a function will be called after a context has cancelled
func Defer(d Doner, cb func()) {
	go func() {
		<-d.Done()
		cb()
	}()
}

// Link ties the lifetime of the Doners to each other.  Link returns a channel
// that fires if ANY of the constituent Doners have fired.
func Link(doners ...Doner) <-chan struct{} {
	c := make(chan struct{})
	cancel := func() { close(c) }

	var once sync.Once
	for _, d := range doners {
		Defer(d, func() { once.Do(cancel) })
	}

	return c
}

// Join returns a channel that receives when all constituent Doners have fired
func Join(doners ...Doner) <-chan struct{} {
	var wg sync.WaitGroup
	wg.Add(len(doners))
	for _, d := range doners {
		Defer(d, wg.Done)
	}

	cq := make(chan struct{})
	go func() {
		wg.Wait()
		close(cq)
	}()
	return cq
}

// FTick calls a function in a loop until the Doner has fired
func FTick(d Doner, f func()) {
	for _ = range Tick(d) {
		f()
	}
}
