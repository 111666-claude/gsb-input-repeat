// Package repeat 是按键连发：带 ID 的按键事件按时间重放、按住状态机、每秒产出预算与到期索引。
package repeat

import "container/heap"

// Event 是一次按键事件。
type Event struct {
	ID   string
	Key  string
	AtMs int
	Down bool
}

// keyState 是一个按键的连发状态。
type keyState struct {
	down    bool
	gen     int
	nextDue int
	count   int
}

// dueEntry 是到期索引里的一项：某键在某刻到期，gen 用来识别过期项。
type dueEntry struct {
	key string
	at  int
	gen int
}

// eventHeap 按（时刻升序、按下先于松开、ID 升序）排序的事件堆。
type eventHeap []Event

func (h eventHeap) Len() int { return len(h) }
func (h eventHeap) Less(i, j int) bool {
	if h[i].AtMs != h[j].AtMs {
		return h[i].AtMs < h[j].AtMs
	}
	if h[i].Down != h[j].Down {
		return h[i].Down
	}
	return h[i].ID < h[j].ID
}
func (h eventHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *eventHeap) Push(x any)   { *h = append(*h, x.(Event)) }
func (h *eventHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	*h = old[:n-1]
	return it
}

// dueHeap 按（到期时刻升序、按键名升序）排序的到期索引。
type dueHeap []dueEntry

func (h dueHeap) Len() int { return len(h) }
func (h dueHeap) Less(i, j int) bool {
	if h[i].at != h[j].at {
		return h[i].at < h[j].at
	}
	return h[i].key < h[j].key
}
func (h dueHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *dueHeap) Push(x any)   { *h = append(*h, x.(dueEntry)) }
func (h *dueHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	*h = old[:n-1]
	return it
}

// Repeater 是按键连发器。
type Repeater struct {
	firstDelay int
	interval   int
	maxRepeats int
	budget     int
	now        int
	ticked     bool
	seen       map[string]struct{}
	pending    eventHeap
	keys       map[string]*keyState
	due        dueHeap
	spent      map[int]int
	dropped    int
	scanned    int
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

// Submit 提交一次按键事件，返回它是否被接受。允许乱序提交；同一个事件 ID 只生效一次；
// 时刻早于已经推进到的位置的事件一律拒绝。
func (r *Repeater) Submit(ev Event) bool {
	if ev.ID == "" || ev.Key == "" {
		return false
	}
	if _, ok := r.seen[ev.ID]; ok {
		return false
	}
	if r.ticked && ev.AtMs < r.now {
		return false
	}
	r.seen[ev.ID] = struct{}{}
	heap.Push(&r.pending, ev)
	return true
}

// Tick 推进到 nowMs，按产出顺序（时刻升序、同刻按键名升序）返回本次产生的重复。
func (r *Repeater) Tick(nowMs int) []string {
	if r.ticked && nowMs < r.now {
		nowMs = r.now
	}
	r.ticked = true
	r.now = nowMs
	out := []string{}
	for {
		evAt, hasEv := r.peekEvent(nowMs)
		dueAt, hasDue := r.peekDue(nowMs)
		if !hasEv && !hasDue {
			break
		}
		if hasEv && (!hasDue || evAt <= dueAt) {
			r.applyEvent(heap.Pop(&r.pending).(Event))
			continue
		}
		if key, ok := r.emitRepeat(heap.Pop(&r.due).(dueEntry)); ok {
			out = append(out, key)
		}
	}
	return out
}

// peekEvent 返回下一个待重放事件的时刻，没有不晚于 nowMs 的事件时返回 false。
func (r *Repeater) peekEvent(nowMs int) (int, bool) {
	if len(r.pending) == 0 || r.pending[0].AtMs > nowMs {
		return 0, false
	}
	return r.pending[0].AtMs, true
}

// peekDue 丢弃过期项后返回下一个到期重复的时刻，没有不晚于 nowMs 的到期项时返回 false。
func (r *Repeater) peekDue(nowMs int) (int, bool) {
	for len(r.due) > 0 {
		top := r.due[0]
		held := r.keys[top.key]
		if held == nil || !held.down || held.gen != top.gen || held.nextDue != top.at {
			heap.Pop(&r.due)
			continue
		}
		if top.at > nowMs {
			return 0, false
		}
		return top.at, true
	}
	return 0, false
}

// applyEvent 重放一次事件：按下把计时与计数重新开始并排入到期索引，松开则停。
func (r *Repeater) applyEvent(ev Event) {
	r.scanned++
	held := r.keys[ev.Key]
	if held == nil {
		held = &keyState{}
		r.keys[ev.Key] = held
	}
	r.scanned++
	held.gen++
	if !ev.Down {
		held.down = false
		return
	}
	held.down = true
	held.count = 0
	held.nextDue = ev.AtMs + r.firstDelay
	heap.Push(&r.due, dueEntry{key: ev.Key, at: held.nextDue, gen: held.gen})
}

// emitRepeat 处理一次到期重复：计入周期计数与每秒预算，放不下的丢弃并计入 Dropped。
func (r *Repeater) emitRepeat(entry dueEntry) (string, bool) {
	held := r.keys[entry.key]
	if held == nil || !held.down || held.gen != entry.gen || held.nextDue != entry.at {
		return "", false
	}
	held.count++
	if r.maxRepeats > 0 && held.count >= r.maxRepeats {
		held.nextDue = 0
	} else if r.interval > 0 {
		held.nextDue = entry.at + r.interval
		heap.Push(&r.due, dueEntry{key: entry.key, at: held.nextDue, gen: held.gen})
	} else {
		held.nextDue = 0
	}
	if r.budget > 0 {
		sec := entry.at / 1000
		if r.spent[sec] >= r.budget {
			r.dropped++
			return "", false
		}
		r.spent[sec]++
	}
	return entry.key, true
}

// Dropped 是累计被每秒预算丢掉的重复次数（规模观测）。
func (r *Repeater) Dropped() int { return r.dropped }

// Scanned 是累计扫描的事件与按键数（规模观测）。
func (r *Repeater) Scanned() int { return r.scanned }
