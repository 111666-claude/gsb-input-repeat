package repeat

import "testing"

func TestIdleTickEmitsNothing(t *testing.T) {
	if keys := New(400, 100, 3, 100).Tick(1000); len(keys) != 0 {
		t.Fatal("没有按下的键时不该产生重复")
	}
}

func TestSubmitRejectsIncompleteEvent(t *testing.T) {
	r := New(400, 100, 3, 100)
	if r.Submit(Event{Key: "a", AtMs: 0, Down: true}) {
		t.Fatal("没有事件 ID 应该被拒绝")
	}
	if r.Submit(Event{ID: "e1", AtMs: 0, Down: true}) {
		t.Fatal("没有按键名应该被拒绝")
	}
}

func TestShortHoldEmitsNothing(t *testing.T) {
	r := New(400, 100, 3, 100)
	r.Submit(Event{ID: "e1", Key: "space", AtMs: 0, Down: true})
	if keys := r.Tick(30); len(keys) != 0 {
		t.Fatal("按得太短不该产生重复")
	}
}

func TestReleasedKeyEmitsNothing(t *testing.T) {
	r := New(400, 100, 3, 100)
	r.Submit(Event{ID: "e1", Key: "space", AtMs: 0, Down: true})
	r.Submit(Event{ID: "e2", Key: "space", AtMs: 10, Down: false})
	if keys := r.Tick(1000); len(keys) != 0 {
		t.Fatal("已经松开的键不该产生重复")
	}
}

func TestCountersStartAtZero(t *testing.T) {
	r := New(400, 100, 3, 100)
	if r.Scanned() != 0 || r.Dropped() != 0 {
		t.Fatal("两个计数器初始都应该是 0")
	}
}
