package repeat

import "testing"

func TestIdleTickEmitsNothing(t *testing.T) {
	if keys := New(400, 100, 3).Tick(1000); len(keys) != 0 {
		t.Fatal("没有按下的键时不该产生重复")
	}
}

func TestShortHoldEmitsNothing(t *testing.T) {
	r := New(400, 100, 3)
	r.Press("space", 0)
	if keys := r.Tick(30); len(keys) != 0 {
		t.Fatal("按得太短不该产生重复")
	}
}

func TestReleasedKeyEmitsNothing(t *testing.T) {
	r := New(400, 100, 3)
	r.Press("space", 0)
	r.Release("space", 10)
	if keys := r.Tick(1000); len(keys) != 0 {
		t.Fatal("已经松开的键不该产生重复")
	}
}

func TestScannedStartsAtZero(t *testing.T) {
	if New(400, 100, 3).Scanned() != 0 {
		t.Fatal("Scanned 初始应该是 0")
	}
}
