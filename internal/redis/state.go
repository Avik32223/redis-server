package redis

import "time"

type State struct {
	data map[string]*stateValue
}

type stateValue struct {
	val       any
	expiresAt time.Time
}

func NewState() State {
	return State{
		data: make(map[string]*stateValue),
	}
}

func (s State) Data() *map[string]*stateValue {
	return &s.data
}
