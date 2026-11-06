# 发布 Checklist

适用组件：`cds-alarm-service`、`cds-snat-configuration`、`cds-pod-alarm`

---

## 0. 组件与镜像对照

| 组件 | 源码入口 | 镜像名 | 部署对象 | 发布文件 |
|---|---|---|---|---|
| cds-alarm-service | `cmd/cds-alarm-service.go` | `cds-alarm-service` | Deployment / kube-system | `releases/cds-alarm-service.yaml` |
| cds-snat-configuration | `cmd/cds-snat-configuration.go` | `cds-snat-configuration` | **DaemonSet** / kube-system | `releases/cds-snat-configuration.yaml` |
| cds-pod-alarm | `cmd/cds-pod-alarm.go` | `cds-pod-alarm` | Deployment / kube-system | `releases/cds-pod-alarm.yaml` |

三个组件共用根目录同一个 `Dockerfile`，通过 `--build-arg bin_file` 区分具体二进制。

> 镜像地址注意：`releases/*.yaml` 里写的是 `capitalonline/<组件>:v2.0.0`（无 registry 前缀），
> 联调用的临时 yaml 里写的是 `registry-bj.capitalonline.net/cck/<组件>:<tag>`。
> **发布前确认发布管线会做地址替换**，否则可能拉不到新镜像。

---

## 1. 通用前置

- [ ] 确认本次发布涉及的环境（dev / pre / 生产）
- [ ] 确认目标集群 K8s 版本（影响节点标签与调度，见 §4.6）
- [ ] 代码已合入发布分支，工作区干净：`git status`
- [ ] 记录当前生产各组件镜像 tag，作为回滚基线
- [ ] 确认目标集群存在 `cds-properties`（`k8s_cluster_id` / `deploy_type`）与 `cck-secrets`（access key）

---

## 2. 镜像构建

### 2.1 构建前

- [ ] 在正确的分支上
- [ ] 确定 tag 规范
  - 联调：`ben-dev-MMDD` 或 `ben-dev-MMDD-序号`
  - 发布：`vX.Y.Z`，**不要用调试 tag 发生产**
- [ ] `docker login` 已登录 registry
- [ ] 确认目标集群架构为 amd64（Dockerfile 中 `GOARCH="amd64"` 为硬编码）

### 2.2 构建并推送

全量构建：

```bash
TAG=v2.1.0
REG=registry-bj.capitalonline.net/cck

for bin in cds-alarm-service cds-snat-configuration cds-pod-alarm; do
  echo "==> $bin:$TAG"
  docker build --build-arg bin_file="$bin" -t "$REG/$bin:$TAG" . || exit 1
  docker push "$REG/$bin:$TAG" || exit 1
done
```

单个组件：

```bash
docker build --build-arg bin_file=cds-pod-alarm \
  -t registry-bj.capitalonline.net/cck/cds-pod-alarm:$TAG .
docker push registry-bj.capitalonline.net/cck/cds-pod-alarm:$TAG
```

### 2.3 构建后验证

- [ ] `docker images | grep <组件>` 确认 tag 与体积正常（Go 静态二进制 + alpine，通常 20~50MB）
- [ ] 在任意节点验证可拉取：

```bash
crictl pull registry-bj.capitalonline.net/cck/<组件>:<TAG>
```

- [ ] 更新工作负载后确认滚动完成：

```bash
kubectl -n kube-system rollout status deploy/cds-alarm-service --timeout=180s
kubectl -n kube-system rollout status ds/cds-snat-configuration --timeout=300s   # DaemonSet 逐节点
kubectl -n kube-system rollout status deploy/cds-pod-alarm --timeout=180s
```

- [ ] 日志确认启动正常，无 `level=fatal`

### 2.4 常见坑

| 问题 | 说明 |
|---|---|
| 没有 `.dockerignore` | 整个仓库（含 `.git`）都会进构建上下文，构建慢；建议补充 |
| 构建阶段执行 `go mod tidy` | 需要能访问依赖源，离线环境会失败；可预置 vendor 或配置 GOPROXY |
| 仅支持 amd64 | 集群若有 arm 节点，需改用 `--platform` 交叉构建 |
| 同名 tag 覆盖 + `imagePullPolicy: Always` | 覆盖推送后必须让 Pod 重建，才会拉取新镜像 |
| DaemonSet 滚动 | `cds-snat-configuration` 逐节点重启，注意选低峰期 |
| 共用 Dockerfile | 漏传 `bin_file` 会导致构建失败或打进错误的二进制 |

---

## 3. 需求一：SNAT 告警升级

### 3.1 影响面

| 组件 | 类型 | 是否需重建 |
|---|---|---|
| `cds-snat-configuration` | DaemonSet | 是（`pkg/snat` + `pkg/monitor`） |
| `cds-alarm-service` | Deployment | 是（`pkg/service`） |
| `cckservice` | 服务 | 是（serializer `max_length`） |
| `cckbackend` | 服务 | 是（serializer `max_length`） |
| `cckfrontend` | 前端 | 是（地址上限 50 → 200） |

### 3.2 发布前

- [ ] cckservice 侧运营评审通过
- [ ] 两个镜像已构建并推送（`cds-snat-configuration`、`cds-alarm-service`，缺一不可）
- [ ] 确认 `snat-configmap` 配置
  - 可选新增 `snat.check.concurrency = 50`（不配默认 50，控制主巡检并发）
  - 复查并发是代码常量 20，不可配置
  - `snat.check.node_ping_dns` 保持 `no`
- [ ] 确认 `releases/cds-alarm-service-ben-dev-1104.yaml` 不随发布包外发

### 3.3 发布顺序

```
1. cckservice + cckbackend   先让服务端接受长 target
2. cds-alarm-service         告警批次计时器修复
3. cds-snat-configuration    DaemonSet 逐节点滚动
4. cckfrontend               最后放开前端输入上限
```

先放开前端会导致：客户端可提交 200 条地址，但服务端 `max_length` 仍是旧值，直接 400 拒绝。

### 3.4 发布后验证

- [ ] 创建 60~200 条地址的监控任务，保存成功
- [ ] 企业微信收到完整地址列表，长消息不再被静默丢弃
- [ ] 添加一个不可达地址，确认「告警」与「恢复」成对出现
- [ ] 日志中不再出现「本轮检测派发耗时超过检查频率」告警
- [ ] DaemonSet 各节点 Ready，Pod 无异常重启

### 3.5 回滚

- [ ] 两个镜像同时回退到基线版本
- [ ] 前后端建议一起回退，避免上下限不一致

---

## 4. 需求二：Terminating Pod 自动删除

### 4.1 发布前

- [ ] 确认生产是否已有 `cds-pod-alarm`，记录当前版本与 ConfigMap 内容
- [ ] 构建正式版本镜像（不要使用 `ben-dev-*` 调试 tag）

### 4.2 升级顺序（必读）

新版将 `pod.monitor.namespaces` 改为**必填**。若老 ConfigMap 中该值为空（旧语义 = 监控全部），
**先换镜像会导致 CrashLoopBackOff**：

```
level=fatal msg="pod.monitor.namespaces is required, configure at least one namespace"
```

必须按以下顺序：

- [ ] 第一步：先修改 ConfigMap，填入真实业务命名空间

```ini
[default]
pod.check.step = 120
pod.monitor.namespaces = <业务命名空间，逗号分隔>   # 必填
pod.pending.threshold.minutes = 20
pod.terminating.threshold.minutes = 10
pod.autodelete.enabled = no                        # 首发保持关闭
pod.autodelete.delay.minutes = 5
```

- [ ] 第二步：删除旧 key `pod.autodelete.namespaces`（新版已忽略，保留会误导）
- [ ] 第三步：再更新 Deployment 镜像

### 4.3 RBAC

- [ ] ClusterRole `cds-pod-alarm-cluster-role` 的 pods verbs 包含 `delete`
- [ ] 发布后验证授权：

```bash
kubectl auth can-i delete pods -n <业务命名空间> \
  --as=system:serviceaccount:kube-system:cds-pod-alarm-sa
```

### 4.4 发布后验证

- [ ] Pod `Running`，日志出现 `cds-pod-alarm start, cluster=xxx, check step=120s`
- [ ] 无 `level=fatal` 或配置类 `level=error`
- [ ] 在测试命名空间造一个 Terminating Pod，确认出现 `[dry-run] 将会强制删除 ...`

### 4.5 dry-run 观察期（3~7 天）

统计未来会被删除的名单（同一 Pod 每轮都会打印，按名字去重后才有意义）：

```bash
kubectl logs -n kube-system deploy/cds-pod-alarm --since=24h \
  | grep '\[dry-run\]' \
  | sed -E 's/.*将会强制删除 ([^，]+)，.*/\1/' \
  | sort | uniq -c | sort -rn
```

统计被跳过原因的分布：

```bash
kubectl logs -n kube-system deploy/cds-pod-alarm --since=24h \
  | grep '跳过自动删除' \
  | sed -E 's/.*跳过自动删除：//' | sort | uniq -c | sort -rn
```

- [ ] 名单中没有意料之外的工作负载；若有，先收窄 `pod.monitor.namespaces`
- [ ] 量级正常（一天几个，而非几十上百）
- [ ] 确认无误后改 `pod.autodelete.enabled = yes`（热更新生效，无需重启）

### 4.6 调度配置核对（当前发布文件有问题）

`releases/cds-pod-alarm.yaml` 中：

```yaml
nodeSelector:
  node-role.kubernetes.io/master: ""              # 1.24+ 已不存在该标签
tolerations:
  - key: node-role.cck.net/infra
  - key: node-role.cck.net/kubernetes.io/master   # 与实际 control-plane 污点不匹配
```

K8s 1.26 集群实测 master 节点：

```
labels: node-role.kubernetes.io/control-plane, node-role.kubernetes.io/infra
taints: node-role.kubernetes.io/control-plane, node-role.cck.net/infra
```

- [ ] 按目标集群版本修正 `nodeSelector` / `tolerations`，否则 Pod 会长期 `Pending`
- [ ] 确认 `DEPLOY_TYPE` / `OPENAPI_IP` 是否由发布管线覆盖；若直接使用仓库 yaml，生产会指向 pre 环境

### 4.7 回滚

| 场景 | 动作 | 生效时间 |
|---|---|---|
| 删除名单不对 / 误删风险 | 改 `pod.autodelete.enabled = no` | 下一周期（≤ check.step），无需重启 |
| 组件自身异常 | 镜像回退到基线版本 | 滚动重启 |
| 完全停用 | `kubectl -n kube-system scale deploy/cds-pod-alarm --replicas=0` | 立即 |

被删除的 Pod 由 Deployment / Job 控制器自动重建，不会造成副本缺失（前提是资格判定生效：
仅处理无状态工作负载、无 PVC、无 finalizer、非系统命名空间的 Pod）。

---

## 5. 发布后 24 小时观察

- [ ] 各组件的日志无异常堆栈
- [ ] SNAT 告警：一线群消息正常，无重复堆积
- [ ] cds-pod-alarm：dry-run 日志持续产出，无 fatal
- [ ] 集群 Terminating Pod 数量趋势（持续增长说明是运行时/网络问题，非本功能可解决）

---

## 6. 回滚速查

```bash
# cds-alarm-service
kubectl -n kube-system set image deploy/cds-alarm-service \
  cds-alarm-service=capitalonline/cds-alarm-service:<基线版本>

# cds-snat-configuration（DaemonSet）
kubectl -n kube-system set image ds/cds-snat-configuration \
  cds-snat-configuration=capitalonline/cds-snat-configuration:<基线版本>

# cds-pod-alarm：优先关开关，再考虑回退镜像
kubectl -n kube-system edit cm pod-alarm-configmap   # pod.autodelete.enabled = no
kubectl -n kube-system set image deploy/cds-pod-alarm \
  cds-pod-alarm=capitalonline/cds-pod-alarm:<基线版本>
```
