package api

import (
	"encoding/json"
	"testing"
)

func TestWSFrameShape(t *testing.T) {
	var got struct {
		Event string         `json:"event"`
		Data  map[string]any `json:"data"`
	}
	if err := json.Unmarshal(wsFrame("sync.users", map[string]any{"users": []int{}}), &got); err != nil {
		t.Fatal(err)
	}
	if got.Event != "sync.users" || got.Data == nil {
		t.Fatalf("unexpected frame %+v", got)
	}
}

func TestWSHubMachineNodeLookup(t *testing.T) {
	h := newWSHub()
	m := &wsConn{send: make(chan []byte, 1), machineID: 7, machineNodes: []int64{10, 11}}
	h.machines[7] = m
	if h.machineFor(11) != m {
		t.Fatal("node 11 should be routed through machine 7")
	}
	if h.machineFor(99) != nil {
		t.Fatal("unknown node must not match")
	}
}

func TestWSEnqueueDropsWhenFull(t *testing.T) {
	c := &wsConn{send: make(chan []byte, 1)}
	if !c.enqueue([]byte("a")) {
		t.Fatal("first message should queue")
	}
	if c.enqueue([]byte("b")) {
		t.Fatal("full queue must drop instead of blocking")
	}
}

func TestContainsID(t *testing.T) {
	if !containsID([]int64{1, 2}, 2) || containsID([]int64{1}, 3) {
		t.Fatal("containsID wrong")
	}
}
