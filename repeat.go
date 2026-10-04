// Package repeat 是按键连发：带 ID 的按键事件按时间重放、按住状态机、每秒产出预算与到期索引。
// 缺陷：按提交顺序重放（乱序被未来事件卡住）、事件不幂等、首次重复用间隔、上限差一、
// 松开后计数不重置、跳帧一次吐积压、每秒预算没做、每帧扫全部按键。
package repeat

// Event 是一次按键事件。
type Event struct {
	ID   string
	Key  string
	AtMs int
	Down bool
}

// state 是一个按键的连发状态。
type state struct {
	lastAt int
	count  int
	down   bool
}

// Repeater 是按键连发器。
type Repeater struct {
	firstDelay int
	interval   int
	maxRepeats int
	budget     int
	events     []Event
	cursor     int
	keys       map[string]*state
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
		keys:       map[string]*state{},
		spent:      map[int]int{},
	}
}

// Submit 提交一次按键事件，返回它是否被接受。
func (r *Repeater) Submit(ev Event) bool {
	if ev.ID == "" || ev.Key == "" {
		return false
	}
	r.events = append(r.events, ev)
	return true
}

// Tick 推进到 nowMs，按产出顺序返回本次产生的重复。
func (r *Repeater) Tick(nowMs int) []string {
	for r.cursor < len(r.events) {
		ev := r.events[r.cursor]
		if ev.AtMs > nowMs {
			break
		}
		r.scanned++
		r.cursor++
		held := r.keys[ev.Key]
		if held == nil {
			held = &state{lastAt: ev.AtMs}
			r.keys[ev.Key] = held
		}
		if ev.Down {
			held.down = true
			held.lastAt = ev.AtMs
		} else {
			held.down = false
		}
	}
	out := []string{}
	for key, held := range r.keys {
		r.scanned++
		if !held.down {
			continue
		}
		delay := r.interval
		if r.maxRepeats > 0 && held.count > r.maxRepeats {
			continue
		}
		if nowMs-held.lastAt < delay {
			continue
		}
		due := (nowMs - held.lastAt) / delay
		for index := 0; index < due; index++ {
			if r.maxRepeats > 0 && held.count > r.maxRepeats {
				break
			}
			out = append(out, key)
			held.count++
		}
		held.lastAt = nowMs
	}
	return out
}

// Dropped 是累计被每秒预算丢掉的重复次数（规模观测）。
func (r *Repeater) Dropped() int { return r.dropped }

// Scanned 是累计扫描的事件与按键数（规模观测）。
func (r *Repeater) Scanned() int { return r.scanned }
