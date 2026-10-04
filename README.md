# input-repeat

按键连发：带 ID 的按键事件按时间重放、按住状态机、每秒产出预算与到期索引，只用 Go 标准库。

```
go run ./cmd/input-repeat --sample=delay
go run ./cmd/input-repeat --sample=cap
go run ./cmd/input-repeat --sample=release
go run ./cmd/input-repeat --sample=outoforder
go run ./cmd/input-repeat --sample=burst
go run ./cmd/input-repeat --sample=idem
go run ./cmd/input-repeat --sample=scan
go test ./...
go vet ./...
```

## 口径（README 为准）

- **事件重放**：事件带 ID、按键、时刻与按下/松开；允许乱序提交，按时刻升序重放，同一时刻按下先于
  松开、再按事件 ID 升序；时刻早于已经推进到的位置的事件一律拒绝。
- **幂等**：同一个事件 ID 只生效一次，重复提交返回 `false` 且不改变任何状态。
- **状态机**：从按下时刻算起满 `firstDelayMs` 产出第一次重复（正好等于算），之后每 `intervalMs`
  一次；同一个按住周期内重复次数不超过 `maxRepeats`；松开或再次按下都把计时与计数重新开始；
  产出时刻必须早于松开时刻。
- **预算配额**：同一整秒内产出的重复次数不超过 `budgetPerSec`，放不下的丢弃并计入 `Dropped`；
  产出按重复时刻升序、同一时刻按按键名升序。
- **到期索引**：`Tick` 只处理本刻到期的按键与事件，不许每帧扫全部按键，`Scanned` 不随按住键数
  乘帧数放大。
- **不变量**：同一组事件无论提交顺序如何，结果一致；松开或到上限的键不再产出；10 万个按键、
  每秒 10 万次 Tick。

## 输出契约（不改格式）

```
keys=..
repeats=..
keys=.. dropped=..
second=.. keys=..
scanned=..
```

## 目录

```
repeat.go             事件重放、状态机、预算与索引
repeat_test.go        单元测试
cmd/input-repeat/     命令行入口
```
