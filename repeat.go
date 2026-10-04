// Package repeat 是按键连发：带 ID 的按键事件按时间重放、按住状态机、每秒产出预算与到期索引。
//
// 事件乱序提交进事件最小堆，重放顺序为时刻升序、同刻按下先于松开、再按 ID 升序；事件生效前
// 先冲刷严格早于该时刻的到期重复，保证产出早于松开。按键只在到期堆顶时被处理，Tick 不扫描
// 全部按键。
package repeat

import "container/heap"

// Event 是一次按键事件。
type Event struct {
	ID   string
	Key  string
	AtMs int
	Down bool
}

// keyState 是一个按键在到期堆中的连发状态。
type keyState struct {
	name    string
	down    bool
	count   int
	nextDue int
	inHeap  bool
	heapIdx int
}

// Repeater 是按键连发器。
type Repeater struct {
	firstDelay int
	interval   int
	maxRepeats int
	budget     int

	events eventQueue
	seen   map[string]struct{}
	keys   map[string]*keyState
	due    dueQueue

	started bool
	now     int
	spent   map[int]int
	dropped int
	scanned int
}

// New 建连发器：firstDelayMs 是首次重复延迟，intervalMs 是重复间隔，maxRepeats 是同一个按住
// 周期内的重复次数上限，budgetPerSec 是同一整秒内最多产出的重复次数。
func New(firstDelayMs, intervalMs, maxRepeats, budgetPerSec int) *Repeater {
	return &Repeater{
		firstDelay: firstDelayMs,
		interval:   intervalMs,
		maxRepeats: maxRepeats,
		budget:     budgetPerSec,
		seen:       map[string]struct{}{},
		keys:       map[string]*keyState{},
		spent:      map[int]int{},
	}
}

// Submit 提交一次按键事件，返回它是否被接受。重复 ID 或时刻早于已推进位置的事件一律拒绝，
// 且拒绝不改变任何状态。
func (r *Repeater) Submit(ev Event) bool {
	if ev.ID == "" || ev.Key == "" {
		return false
	}
	if _, dup := r.seen[ev.ID]; dup {
		return false
	}
	if r.started && ev.AtMs < r.now {
		return false
	}
	r.seen[ev.ID] = struct{}{}
	heap.Push(&r.events, ev)
	return true
}

// Tick 推进到 nowMs，按产出顺序（重复时刻升序、同刻按键名升序）返回本次产生的重复。
func (r *Repeater) Tick(nowMs int) []string {
	if r.started && nowMs < r.now {
		return nil
	}
	r.started = true
	out := []string{}
	for r.events.Len() > 0 && r.events[0].AtMs <= nowMs {
		ev := heap.Pop(&r.events).(Event)
		// 事件生效前，先产出严格早于事件时刻的重复，保证产出早于松开/再按。
		r.flushDue(ev.AtMs, true, &out)
		r.apply(ev)
	}
	r.flushDue(nowMs, false, &out)
	r.now = nowMs
	return out
}

// apply 让一个已排序事件生效：按下重置计时与计数，松开摘除到期索引。
func (r *Repeater) apply(ev Event) {
	r.scanned++
	k := r.keys[ev.Key]
	if ev.Down {
		if k == nil {
			k = &keyState{name: ev.Key, heapIdx: -1}
			r.keys[ev.Key] = k
			r.scanned++
		}
		k.down = true
		k.count = 0
		k.nextDue = ev.AtMs + r.firstDelay
		if k.inHeap {
			heap.Fix(&r.due, k.heapIdx)
		} else {
			heap.Push(&r.due, k)
		}
		return
	}
	if k != nil && k.down {
		k.down = false
		if k.inHeap {
			heap.Remove(&r.due, k.heapIdx)
			k.inHeap = false
		}
	}
}

// flushDue 处理到期堆中早于（strict）或不晚于 limit 的按键，每个键只在本刻到期时被处理一次。
func (r *Repeater) flushDue(limit int, strict bool, out *[]string) {
	for r.due.Len() > 0 {
		k := r.due[0]
		if strict {
			if k.nextDue >= limit {
				return
			}
		} else if k.nextDue > limit {
			return
		}
		heap.Pop(&r.due)
		due := k.nextDue
		second := due / 1000
		if r.spent[second] < r.budget {
			*out = append(*out, k.name)
			r.spent[second]++
		} else {
			r.dropped++
		}
		k.count++
		if k.count < r.maxRepeats {
			k.nextDue = due + r.interval
			heap.Push(&r.due, k)
		}
	}
}

// Dropped 是累计被每秒预算丢掉的重复次数（规模观测）。
func (r *Repeater) Dropped() int { return r.dropped }

// Scanned 是累计扫描的事件与按键数（规模观测）：每个事件与每个进入到期索引的按键各计一次，
// 不随按住键数乘帧数放大。
func (r *Repeater) Scanned() int { return r.scanned }

// eventQueue 按时刻升序、同刻按下先于松开、再按事件 ID 升序排列。
type eventQueue []Event

func (q eventQueue) Len() int { return len(q) }

func (q eventQueue) Less(i, j int) bool {
	a, b := q[i], q[j]
	if a.AtMs != b.AtMs {
		return a.AtMs < b.AtMs
	}
	if a.Down != b.Down {
		return a.Down
	}
	return a.ID < b.ID
}

func (q eventQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }

func (q *eventQueue) Push(x any) { *q = append(*q, x.(Event)) }

func (q *eventQueue) Pop() any {
	old := *q
	n := len(old)
	ev := old[n-1]
	*q = old[:n-1]
	return ev
}

// dueQueue 是按键到期最小堆：到期时刻升序、同刻按键名升序；每个按住键只有一个索引项。
type dueQueue []*keyState

func (q dueQueue) Len() int { return len(q) }

func (q dueQueue) Less(i, j int) bool {
	a, b := q[i], q[j]
	if a.nextDue != b.nextDue {
		return a.nextDue < b.nextDue
	}
	return a.name < b.name
}

func (q dueQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].heapIdx = i
	q[j].heapIdx = j
}

func (q *dueQueue) Push(x any) {
	k := x.(*keyState)
	k.heapIdx = len(*q)
	k.inHeap = true
	*q = append(*q, k)
}

func (q *dueQueue) Pop() any {
	old := *q
	n := len(old)
	k := old[n-1]
	old[n-1] = nil
	*q = old[:n-1]
	k.inHeap = false
	k.heapIdx = -1
	return k
}
