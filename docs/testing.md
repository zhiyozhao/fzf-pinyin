[开发指南](../CONTRIBUTING.md)

# 测试与性能测量

面向修改匹配算法、缓存或终端行为的开发者。常规入口为 `make -f Makefile.fzfp check`；以下说明要验证什么，以及何时运行额外套件。

## 匹配与 CLI

- 无 Han 候选的分数、范围、位置数组，以及完整 CLI 输出、排序、stderr 和退出码与固定上游参考一致；不能只按查询是否为英文判断兼容路径。
- 合法拼写的非空前缀持续召回；不合法的音节中间截断不能意外命中。
- 小输入用独立穷举路径核对最高分，避免以实现本身生成预期。
- 相同拼写证据下，增加间隔不会提高分数；边界优势和不同拼写分段应分别考虑。
- 同字母、同边界和连续块下，字面连续命中恰好比拼音高 1 分，连续拼音胜过插入普通字母 gap 的字面命中；跨字、跨脚本继承连续块奖励。全拼、缩写及末字前缀每字成本均为 0，只在用了拼音的整条匹配扣 1 分；含汉字的纯字面路径及无拼音字母查询不扣。穷举 oracle 按整条路径是否用了拼音独立计分。用三万行合成路径/标题检查全拼和缩写的完整排序，反例及边界条件见[评分预期](design.md#排序预期)。
- 原文位置、高亮、字段、否定、大小写及末字前缀共同工作。
- 缓存及逐字输入结果与全量重算一致；并发调用无数据竞争。

`check` 包含 Go / CLI / PTY、race 和安装器回归；新增规则应有独立的正反例，测试数量不能代替行为保证。`make -f Makefile.fzfp fuzz` 运行前缀召回 fuzzing，失败输入应保留为可复现回归。

## 性能测量

运行 `make -f Makefile.fzfp bench`。它使用固定生成的 100,000 行语料，分别覆盖无 Han 和中英混合输入；完整 CLI 测量包含启动、字典初始化、读取、匹配、排序和输出，每组预热后交替测量多次，报告中位数。库基准另测热调用，不包含进程启动。

比较前后版本时保持机器、Go 版本、输入、命中行数和位置回溯设置一致。不同命中集合的耗时比不能直接当成算法开销，单行库耗时也不能推算完整 CLI 延迟。对超长行、长查询和多音字密集内容按实际规模补测。

日志和临时报告放在 `Artifacts/`。发布测量结果时注明环境、输入、方法和范围；不要把一次测量当成所有机器上的速度保证。

`internal/fixture/perf.go` 使用种子 20261123 生成路径/标题、超长行、48 字母查询、多音字和纯英文语料。`go run ./cmd/perf -bins baseline=/tmp/before,optimized=/tmp/after -compare` 对真实 `--filter` 进程预热后交替测量 21 次，输出中位数、nearest-rank p95 和输出 SHA-256。用各版本 `./cmd/perfcli` 的构建传入 `-alloc-bins baseline=/tmp/before-alloc,optimized=/tmp/after-alloc`，单独测量完整过滤进程的累计 Go 分配（含初始化）；正常耗时测量不含分配探针。纯英文对照在同一命令中加入固定上游的正常/探针程序。

`FZFP_PERF=1 go test -run '^TestPerfQuantiles$' -v ./src/pinyin` 对同一语料做库级全量扫描，与保留的 `76553e1d` 实现交替测量 21 次，记录中位数、p95 和平均分配。`FZFP_PERF_CASE=long` 可筛选场景。`go test -run '^$' -bench BenchmarkPerfScan -benchmem ./src/pinyin` 提供标准 Go benchmark（每 op 为一遍完整语料）；原有 `BenchmarkMatch` 每 op 为一行，两者数字不可直接比较。性能测量应独占运行，避免与 race、profile 或另一轮测量争用 CPU。

`FZFP_PERF=1 go test -run '^TestPerfLiteralPass$' -v ./src/pinyin` 用 Wrap 原有条件在计时外筛出需要补算字面的候选，只测额外一遍 DP 的调用数、分位数及分配；不是关闭补算后比较不同结果。原 24 组固定语料不变，另加 100k 行 `田abc`/`国中a` 和长字面位置组覆盖补算分支。

`go test -count=1 -run '^TestLegacyDifferential$' -v ./src/pinyin` 在三个独立 scheme 进程内对 76553e1d 的全路径 DP、纯字面 DP 和完整 Wrap 比较 score、起止和 positions（含 nil、顺序）；两套 DP 都使用当前输入法拼写集合。默认规模使用种子 20261123 的前 2,000 组随机用例，对每个固定性能场景均匀抽取最多 128 行（含首尾和未命中行）。固定语料按抽样序号交替 forward/backward；每个用例均比较带/不带位置及允许/禁用拼音边。长行、48 字母查询、多音字、英文、字面补算、长间隔回溯等类别全部保留；长行与 scratch 容量回归保持原始长度，所有单行场景另跑六种模式、两个方向。随机用例继续覆盖六种模式、大小写和归一化组合。

**改动拼音匹配代码时，合并前必须本地运行一次完整规模并记录结果**：

```sh
FZFP_FULL_DIFF=1 go test -timeout 20m -count=1 -run '^TestLegacyDifferential$' -v ./src/pinyin
```

完整规模使用全部 60,000 组随机用例和所有固定语料候选，不按命中结果筛选，合计千万级比对；日志列出每个场景实际/总行数以及每种 scheme 的比对数。这里的本地 `-timeout 20m` 不修改默认 CI 超时。race 编译仍跳过这项串行差分，其他 race 检查不变。

设置 `FZFP_BASELINE_BIN`（旧 zeroFinal 规则的 76553e1d 或其后 main 构建）、`FZFP_BIN` 和 `FZF_REFERENCE_BIN` 后，`go test -run '^TestPerfLegacyCLI$' -v ./integration` 对拼写行为未变的查询逐字节比较完整排序输出、stderr 和退出码。受规则删除影响的 `nv` 改由明确差异断言覆盖：旧版可用 `n+v` 找到 `n乐`，新版只保留女的 nv；完整新规则另由 IME 正反例测试验证。

## 完整终端对照

`make -f Makefile.fzfp e2e` 先构建两份程序，再对固定上游和 fzfp 运行同一套 Ruby/tmux 断言。参考程序来自 `.cache/upstream`，不使用 fork 根目录默认 Makefile 构建。该套件需要额外依赖且可能运行数分钟，因此与常规 `check` 分开；匹配或终端行为变化时应运行。

需要 Ruby 3.4+、tmux、checkout 内 `.cache/gems` 中的 minitest 5.25.4。floating-pane 覆盖需要 tmux 3.7+。`E2E_RUBY` 默认取 PATH 中的 ruby；macOS 系统 Ruby 通常过旧。用 `E2E_TMUX` 指定 tmux，非 Apple Silicon Homebrew 环境尤其需要检查其路径。

选择兼容解释器并仅在 checkout 安装依赖：

```sh
# 先将 E2E_RUBY 设为兼容 Ruby 的可执行路径并导出。
GEM_HOME="$PWD/.cache/gems" GEM_PATH="$PWD/.cache/gems" \
  "$E2E_RUBY" -S gem install --no-document minitest -v 5.25.4
make -f Makefile.fzfp e2e
```

若使用 Ruby 4，需要为 minitest 5.25.4 加 `--force` 以绕过其 Ruby <4.0 元数据限制，并核对实际运行结果；这不代表依赖官方支持 Ruby 4。不要修改全局 gems。

套件覆盖 core、exec、filter、layout、preview、raw、server 和 tmux，不含 shell 集成和 Vim。两轮 fixture 内虽然都叫 `fzf`，实际分别指向被测程序，不改断言或版本输出。

测试使用独立 tmux socket、空配置、临时 HOME/配置/历史/目录和独占端口，默认 80×24。环境适配仅在临时上游副本进行，结束后关闭所属进程并清理目录。

报告放在 `Artifacts/e2e/`，可用 `E2E_OUTPUT` 改目录。逐用例 JSONL、注册表、终端日志和汇总用于核对完整性；`comparison.json` 列出两轮状态差异，新增失败、跳过或缺失使目标失败。共享错误仍应逐项检查，不能凭总数相同判断兼容。

布局组合矩阵可能受 80×24 下可见列表行数限制：断言访问第二个条目时，预览可能只留下一个条目。遇到这种共享错误时检查终端尺寸、原始日志和两边的具体断言；不能把未完成的矩阵算作已通过。`E2E_LONGTEST=0` 可关闭长矩阵，但须明确报告覆盖缩小。
