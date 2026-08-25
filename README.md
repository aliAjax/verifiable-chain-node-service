# Verifiable Chain Node

纯 Go 的可验证区块链节点与轻客户端状态同步平台示例。服务不执行脚本，交易只改变受限账户/键值状态。

## 启动

```bash
go run ./cmd/node
curl localhost:8080/healthz
```

## 主要 API

- `GET /api/v1/networks`
- `GET /api/v1/peers`
- `POST /api/v1/transactions`
- `GET /api/v1/mempool`
- `POST /api/v1/blocks/mine`
- `GET /api/v1/blocks/{height}`
- `POST /api/v1/sync`
- `POST /api/v1/snapshots`
- `POST /api/v1/snapshots/{id}/verify`
- `POST /api/v1/rollback-plan`
- `GET /api/v1/light/checkpoints`
- `GET /api/v1/proofs/{key}`
- `GET /api/v1/consensus/views`

详见 `docs/openapi.yaml`、`proto/node.proto` 与 `deployments/docker-compose.yml`。
