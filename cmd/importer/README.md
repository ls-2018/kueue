# Kueue 导入工具

一个能够将现有 Pod 导入到 Kueue 的工具。

## 集群设置

导入器应在已定义 Kueue CRD 的集群中运行，并且 `kueue-controller-manager` 未运行或已禁用 `pod` 集成框架。详情请查阅 Kueue 的[安装指南](https://kueue.sigs.k8s.io/docs/installation/)和[运行原生 Pod](https://kueue.sigs.k8s.io/docs/tasks/run_plain_pods/#before-you-begin)。

要成功导入，所有相关的 Kueue 对象（LocalQueue、ClusterQueue 和 ResourceFlavor）都需要在集群中创建，导入器的检查阶段会检查并列出缺失的对象。

## 构建

在 kueue 源码根目录下运行：

```bash
make importer-build
```

## 用法

该命令会使用系统默认的 kubectl 配置。更多关于如何配置访问多个集群的信息，请查阅 [kubectl 文档](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)。

### 检查

导入器会执行以下检查：

- 至少提供一个 `namespace`。
- 每个 Pod 都有到 LocalQueue 的映射。
- 目标 LocalQueue 存在。
- 参与导入的 LocalQueue 使用已存在的 ClusterQueue。
- 参与的 ClusterQueue 至少有一个 ResourceGroup 使用已存在的 ResourceFlavor。导入器在为创建的 Workload 创建准入时会使用该 ResourceFlavor。

Pod 到 LocalQueue 的映射有两种方式：

#### 简单映射

通过指定标签名和任意数量的 `<标签值>=<LocalQueue 名称>` 作为命令行参数，例如：  
`--queuelabel=src.lbl --queuemapping=src-val=user-queue,src-val2=user-queue2`

#### 高级映射

通过 `--queuemapping-file` 参数提供一个 yaml 映射文件，内容示例：

```yaml
- match:
    labels:
      src.lbl: src-val
  toLocalQueue: user-queue
- match:
    priorityClassName: p-class
    labels:
      src.lbl: src-val2
      src2.lbl: src2-val
  toLocalQueue: user-queue2
- match:
    labels:
      src.lbl: src-val3
  skip: true
```

- 匹配规则中如果没有 `priorityClassName`，则忽略 Pod 的 `priorityClassName`，如果有多个 `label: value`，则都需匹配。
- 规则按顺序依次评估。
- `skip: true` 可用于忽略匹配该规则的 Pod。

#### 其他参数

构建可执行文件后，可通过以下命令获取所有支持的参数：

```bash
./bin/importer import help
```

示例输出：

```bash
Usage:
  importer import [flags]

Flags:
      --add-labels stringToString     为导入的 Pod 和创建的 Workload 添加额外的 label=value（默认 []）
      --burst int                     客户端 Burst，详见 https://kubernetes.io/docs/reference/config-api/apiserver-eventratelimit.v1alpha1/#eventratelimit-admission-k8s-io-v1alpha1-Limit（默认 50）
  -c, --concurrent-workers uint       并发导入 worker 数量（默认 8）
      --dry-run                       仅检查配置，不实际导入（默认 true）
  -h, --help                          显示帮助
  -n, --namespace strings             目标命名空间（至少需提供一个）
      --qps float32                   客户端 QPS，详见上文链接（默认 50）
      --queuelabel string             用于标识目标 LocalQueue 的标签
      --queuemapping stringToString   从 "queuelabel" 标签值到 LocalQueue 名称的映射（默认 []）
      --queuemapping-file string      包含额外映射的 yaml 文件

Global Flags:
  -v, --verbose count   日志详细程度（可多次指定以增加日志级别）
```

- 至少需要指定一个 `namespace`
- `queuelabel` 和 `queuemapping` 应同时使用
- `queuemapping` 和 `queuemapping-file` 必须且只能指定一个

### 导入

运行导入器时，如果指定了 `--dry-run=false`，则对于每个选中的 Pod，导入器会：

- 更新 Pod 的 Kueue 相关标签
- 创建与 Pod 关联的 Workload
- 批准该 Workload

### 示例

#### 简单映射

```bash
./bin/importer import -n ns1,ns2 --queuelabel=src.lbl --queuemapping=src-val=user-queue,src-val2=user-queue2 --dry-run=false
```

会将命名空间 `ns1` 或 `ns2` 中，`src.lbl` 标签值为 `src-val` 或 `src-val2` 的 Pod，分别导入到 LocalQueue `user-queue` 或 `user-queue2`。

#### 高级映射

使用映射文件：

```yaml
- match:
    labels:
      src.lbl: src-val
  toLocalQueue: user-queue
- match:
    priorityClassName: p-class
    labels:
      src.lbl: src-val2
      src2.lbl: src2-val
  toLocalQueue: user-queue2
```

```bash
./bin/importer import -n ns1,ns2  --queuemapping-file=<mapping-file-path> --dry-run=false
```

会将命名空间 `ns1` 或 `ns2` 中，`src.lbl=src-val` 的 Pod 导入到 LocalQueue `user-queue`（不考虑 priorityClassName），以及 `src.lbl=src-val2`、`src2.lbl=src2-val` 且 `priorityClassName=p-class` 的 Pod 导入到 `user-queue2`。

### 集群内运行

`cmd/importer/run-in-cluster` 提供了在集群内运行导入器所需的 kustomize 清单。

使用方法：

1. （可选）构建镜像

运行以下命令构建包含导入器的最小镜像：

```bash
make importer-image
```

确保镜像可被集群访问。

2. 设置使用的镜像

```bash
(cd cmd/importer/run-in-cluster && kustomize edit set image importer=<image:tag>)
```
你可以使用上一步构建的镜像，或使用官方发布的镜像，例如：  
`us-central1-docker.pkg.dev/k8s-staging-images/kueue/importer:main-latest`

3. 根据需要更新 `cmd/importer/run-in-cluster/importer.yaml` 中的导入器参数

注意：`dry-run` 默认设置为 `false`

4. 更新 `cmd/importer/run-in-cluster/mapping.yaml` 中的映射配置

5. （可选）本地 dry-run 检查配置，例如：

```bash
./bin/importer --dry-run=true <your-custom-flags> --queuemapping-file=cmd/importer/run-in-cluster/mapping.yaml
```

6. 部署配置：

```bash
kubectl apply -k cmd/importer/run-in-cluster/
```

并查看日志：

```bash
kubectl -n kueue-importer logs kueue-importer -f
```
