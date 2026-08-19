# Bug reproduction

## Bug

同一晶圆批次被并发调度时可能出现多个成功租约；事件发布因请求取消失败时，已取得的租约也可能残留。

## Trigger

在项目根目录分别执行以下定向验证命令：

```bash
go test -race ./internal/lots -run '^TestLeaseCoordinatorConcurrentSingleWinner$' -count=1
```

```bash
go test -race ./internal/lots -run '^TestLeaseCoordinatorCanceledPublishStillRollsBack$' -count=1
```

## Error information

埋错版本会在并发租约测试中出现 race 或多个调度器同时成功；取消发布测试会报告租约未被释放。
