// Copyright (c) 2025 Stefano Scafiti
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
package sync

import (
	"sync"
)

type PauseGate struct {
	mu      sync.Mutex
	counter int64
	cond    *sync.Cond
}

func NewPauseGate() *PauseGate {
	g := &PauseGate{}
	g.cond = sync.NewCond(&g.mu)
	return g
}

func (g *PauseGate) Pause() {
	g.mu.Lock()
	g.counter++
	g.mu.Unlock()
}

func (g *PauseGate) Resume() {
	g.mu.Lock()
	g.counter--
	if g.counter < 0 {
		panic("Resume() called without any previous call to Pause()")
	}
	g.cond.Signal()
	g.mu.Unlock()
}

func (g *PauseGate) WaitResume() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.counter == 0 {
		return
	}

	for g.counter > 0 {
		g.cond.Wait()
	}
}
