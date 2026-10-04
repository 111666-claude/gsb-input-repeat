# input-repeat

按键连发：首次延迟、重复间隔、次数上限、释放重置与到期索引，只用 Go 标准库。

```
go run ./cmd/input-repeat --sample=delay
go run ./cmd/input-repeat --sample=cap
go run ./cmd/input-repeat --sample=release
go run ./cmd/input-repeat --sample=burst
go run ./cmd/input-repeat --sample=scan
go test ./...
go vet ./...
```

## 口径（README 为准）

- **首次重复**：从按下时刻算起，按住满 `firstDelayMs` 产出第一次重复（正好等于算），
  之后每 `intervalMs` 一次，间隔从本次产出时刻重新算。
- **上限**：同一个按住周期内重复次数不超过 `maxRepeats`；`maxRepeats` 不大于 0 表示不连发。
- **释放重置**：`Release` 之后该键的计时与计数全部清空，再按下重新从首次延迟开始。
- **按帧摊开**：单次 `Tick` 每个到期按键最多产出一件重复，一次跳帧不许把积压的重复一口气吐出来。
- **不变量**：松开或到上限的键不再产出；返回结果按按键名升序；同一串按下、松开与 Tick 序列
  重复回放结果相同。
- **代价**：`Tick` 只处理本刻到期的键，不许每帧扫全部按住键，`Scanned` 不随按住键数乘帧数放大；
  10 万个按键、每秒 10 万次 Tick。

## 输出契约（不改格式）

```
keys=..
repeats=..
scanned=..
```

## 目录

```
repeat.go             首次延迟、间隔、上限与释放重置
repeat_test.go        单元测试
cmd/input-repeat/     命令行入口
```
