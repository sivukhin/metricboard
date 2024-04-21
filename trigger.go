package main

import (
	"sync"
)

type Trigger struct {
	sync.Mutex
	countdown int
	activated bool
	callback  func()
}

func NewTrigger(callback func()) *Trigger {
	return &Trigger{callback: callback}
}

func (t *Trigger) Add() {
	t.Lock()
	t.countdown++
	t.Unlock()
}

func (t *Trigger) Done() {
	t.Lock()
	t.countdown--
	countdown := t.countdown
	activated := t.activated
	t.Unlock()

	if countdown == 0 && activated {
		t.callback()
	}
}

func (t *Trigger) Activate() {
	t.Lock()
	t.activated = true
	countdown := t.countdown
	activated := t.activated
	t.Unlock()

	if countdown == 0 && activated {
		t.callback()
	}
}
