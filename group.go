package ssh

import (
	"fmt"
	"sync"
)

type Group struct {
	err error
	wg  sync.WaitGroup
	mux sync.Mutex
}

func WithContext() *Group {
	return &Group{}
}

func (g *Group) Go(f func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := f(); err != nil {
			g.mux.Lock()
			g.err = fmt.Errorf("%w", err)
			g.mux.Unlock()
		}
	}()
}
func (g *Group) Wait() {
	g.wg.Wait()

}

func (g *Group) Error() error {
	g.mux.Lock()
	defer g.mux.Unlock()
	return g.err
}
