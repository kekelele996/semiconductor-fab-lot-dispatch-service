# Bug reproduction

## Bug

目标缺陷会在公开调度链路的并发、取消、错误传播或可变状态场景中暴露。

## Trigger

在项目根目录依次执行以下定向验证命令：

```bash
go test -race ./internal/tools -run '^TestReservationServiceOverlappingRoutesAreAtomic$' -count=1
```

```bash
go test -race ./internal/tools -run '^TestReservationServiceCanceledPublishReleasesWholeRoute$' -count=1
```

## Error information

埋错版本的目标测试会稳定失败；Gold 修复版本应全部通过。
