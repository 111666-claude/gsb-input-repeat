// Package repeat 是按键连发：首次延迟、重复间隔、次数上限、释放重置与到期索引。
// 缺陷：首次重复用间隔而不是首次延迟、上限差一、释放不重置计时、一次跳帧把积压的重复
// 一口气吐出来、每次 Tick 扫全部按住键。
package repeat

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
	held       map[string]*state
	scanned    int
}

// New 建连发器：firstDelayMs 是首次重复延迟，intervalMs 是重复间隔，
// maxRepeats 是同一个按住周期内的重复次数上限。
func New(firstDelayMs, intervalMs, maxRepeats int) *Repeater {
	return &Repeater{
		firstDelay: firstDelayMs,
		interval:   intervalMs,
		maxRepeats: maxRepeats,
		held:       map[string]*state{},
	}
}

// Press 记录一次按下。
func (r *Repeater) Press(key string, atMs int) {
	if held := r.held[key]; held != nil {
		held.down = true
		return
	}
	r.held[key] = &state{lastAt: atMs, down: true}
}

// Release 记录一次松开。
func (r *Repeater) Release(key string, atMs int) {
	if held := r.held[key]; held != nil {
		held.down = false
	}
}

// Tick 推进到 nowMs，返回本次产生重复的按键，按名字升序。
func (r *Repeater) Tick(nowMs int) []string {
	out := []string{}
	for key, held := range r.held {
		r.scanned++
		if !held.down {
			continue
		}
		delay := r.interval
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
		held.lastAt = nowMs - (nowMs-held.lastAt)%delay
	}
	return out
}

// Scanned 是累计扫描的按键数（规模观测）。
func (r *Repeater) Scanned() int { return r.scanned }
