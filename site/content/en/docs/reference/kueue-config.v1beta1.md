---
title: Kueue Configuration API
content_type: tool-reference
package: /v1beta1
auto_generated: true
description: Generated API reference documentation for Kueue Configuration.
---


## Resource Types 


- [Configuration](#Configuration)
  
    
    

## `AdmissionFairSharing`     {#AdmissionFairSharing}
    

**Appears in:**




<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>usageHalfLifeTime</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#duration-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Duration</code></a>
</td>
<td>
   <p>usageHalfLifeTime 表示当前使用量在达到一半后衰减的时间。
如果设置为 0，则使用量将立即重置为 0。</p>
</td>
</tr>
<tr><td><code>usageSamplingInterval</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#duration-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Duration</code></a>
</td>
<td>
   <p>usageSamplingInterval 表示 Kueue 更新 FairSharingStatus 中 consumedResources 的频率。
默认为 5min。</p>
</td>
</tr>
<tr><td><code>resourceWeights</code> <B>[Required]</B><br/>
<code>map[ResourceName]float64</code>
</td>
<td>
   <p>resourceWeights 为资源分配权重，然后用于计算 LocalQueue 的资源使用情况和排序 Workloads。
默认为 1。</p>
</td>
</tr>
</tbody>
</table>

## `ClientConnection`     {#ClientConnection}
    

**Appears in:**




<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>qps</code> <B>[Required]</B><br/>
<code>float32</code>
</td>
<td>
   <p>QPS 控制允许 K8S API server 连接的每秒查询数。</p>
</td>
</tr>
<tr><td><code>burst</code> <B>[Required]</B><br/>
<code>int32</code>
</td>
<td>
   <p>Burst 允许客户端在超出其速率时积累额外的查询。</p>
</td>
</tr>
</tbody>
</table>

## `ClusterQueueVisibility`     {#ClusterQueueVisibility}
    

**Appears in:**

- [QueueVisibility](#QueueVisibility)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>maxCount</code> <B>[Required]</B><br/>
<code>int32</code>
</td>
<td>
   <p>MaxCount 表示在集群队列状态中暴露的最大待处理工作负载数量。
当值设置为 0 时，则禁用集群队列可见性更新。
最大值为 4000。
默认为 10。</p>
</td>
</tr>
</tbody>
</table>

## `Configuration`     {#Configuration}
    


<p>Configuration 是 kueueconfigurations API 的 Schema</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>namespace</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>Namespace 是 kueue 部署所在的命名空间。它作为 webhook Service 的 DNSName 的一部分。
如果未设置，则从文件 /var/run/secrets/kubernetes.io/serviceaccount/namespace 获取值。
如果文件不存在，默认值为 kueue-system。</p>
</td>
</tr>
<tr><td><code>ControllerManager</code> <B>[Required]</B><br/>
<a href="#ControllerManager"><code>ControllerManager</code></a>
</td>
<td>(Members of <code>ControllerManager</code> are embedded into this type.)
   <p>ControllerManager 返回控制器的配置</p>
</td>
</tr>
<tr><td><code>manageJobsWithoutQueueName</code> <B>[Required]</B><br/>
<code>bool</code>
</td>
<td>
   <p>ManageJobsWithoutQueueName 控制 Kueue 是否管理未设置标签 kueue.x-k8s.io/queue-name 的作业。
如果设置为 true，则这些作业将被挂起，除非分配了队列并最终被接收，否则永远不会启动。这也适用于启动 kueue 控制器之前创建的作业。
默认为 false，因此这些作业不受管理，如果创建时未挂起，将立即启动。</p>
</td>
</tr>
<tr><td><code>managedJobsNamespaceSelector</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#labelselector-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector</code></a>
</td>
<td>
   <p>ManagedJobsNamespaceSelector 提供基于命名空间的机制以使作业不被 Kueue 管理。</p>
<p>对于基于 Pod 的集成（pod、deployment、statefulset 等），只有命名空间匹配 ManagedJobsNamespaceSelector 的作业才有资格被 Kueue 管理。
在不匹配命名空间中的 Pod、deployment 等永远不会被 Kueue 管理，即使它们有 kueue.x-k8s.io/queue-name 标签。
这种强豁免确保 Kueue 不会干扰系统命名空间的基本操作。</p>
<p>对于所有其他集成，ManagedJobsNamespaceSelector 仅通过调节 ManageJobsWithoutQueueName 的效果提供较弱的豁免。
对于这些集成，具有 kueue.x-k8s.io/queue-name 标签的作业将始终由 Kueue 管理。没有该标签的作业仅在 ManageJobsWithoutQueueName 为 true 且命名空间匹配 ManagedJobsNamespaceSelector 时才由 Kueue 管理。</p>
</td>
</tr>
<tr><td><code>internalCertManagement</code> <B>[Required]</B><br/>
<a href="#InternalCertManagement"><code>InternalCertManagement</code></a>
</td>
<td>
   <p>InternalCertManagement 是 internalCertManagement 的配置</p>
</td>
</tr>
<tr><td><code>waitForPodsReady</code> <B>[Required]</B><br/>
<a href="#WaitForPodsReady"><code>WaitForPodsReady</code></a>
</td>
<td>
   <p>WaitForPodsReady 是为作业提供基于时间的全有或全无调度语义的配置，
通过确保所有 pod 在指定时间内就绪（运行并通过就绪探针）。如果超时，则驱逐该工作负载。</p>
</td>
</tr>
<tr><td><code>clientConnection</code> <B>[Required]</B><br/>
<a href="#ClientConnection"><code>ClientConnection</code></a>
</td>
<td>
   <p>ClientConnection 提供 Kubernetes API server 客户端的其他配置选项。</p>
</td>
</tr>
<tr><td><code>integrations</code> <B>[Required]</B><br/>
<a href="#Integrations"><code>Integrations</code></a>
</td>
<td>
   <p>Integrations 提供 AI/ML/Batch 框架集成（包括 K8S job）的配置选项。</p>
</td>
</tr>
<tr><td><code>queueVisibility</code> <B>[Required]</B><br/>
<a href="#QueueVisibility"><code>QueueVisibility</code></a>
</td>
<td>
   <p>QueueVisibility 用于暴露关于顶部待处理工作负载的信息。
已废弃：该字段将在 v1beta2 移除，请使用 VisibilityOnDemand
(https://kueue.sigs.k8s.io/docs/tasks/manage/monitor_pending_workloads/pending_workloads_on_demand/)
代替。</p>
</td>
</tr>
<tr><td><code>multiKueue</code> <B>[Required]</B><br/>
<a href="#MultiKueue"><code>MultiKueue</code></a>
</td>
<td>
   <p>MultiKueue 控制 MultiKueue AdmissionCheck Controller 的行为。</p>
</td>
</tr>
<tr><td><code>fairSharing</code> <B>[Required]</B><br/>
<a href="#FairSharing"><code>FairSharing</code></a>
</td>
<td>
   <p>FairSharing 控制集群范围内的公平共享语义。</p>
</td>
</tr>
<tr><td><code>admissionFairSharing</code> <B>[Required]</B><br/>
<a href="#AdmissionFairSharing"><code>AdmissionFairSharing</code></a>
</td>
<td>
   <p>admissionFairSharing 表示开启 <code>AdmissionTime</code> 模式下的 FairSharing 配置</p>
</td>
</tr>
<tr><td><code>resources</code> <B>[Required]</B><br/>
<a href="#Resources"><code>Resources</code></a>
</td>
<td>
   <p>Resources 提供处理资源的其他配置选项。</p>
</td>
</tr>
<tr><td><code>featureGates</code> <B>[Required]</B><br/>
<code>map[string]bool</code>
</td>
<td>
   <p>FeatureGates 是特性名称到布尔值的映射，允许覆盖特性的默认启用状态。该映射不能与通过命令行参数 &quot;--feature-gates&quot; 传递特性列表同时使用。</p>
</td>
</tr>
<tr><td><code>objectRetentionPolicies</code><br/>
<a href="#ObjectRetentionPolicies"><code>ObjectRetentionPolicies</code></a>
</td>
<td>
   <p>ObjectRetentionPolicies 提供 Kueue 管理对象自动删除的配置选项。nil 值禁用所有自动删除。</p>
</td>
</tr>
</tbody>
</table>

## `ControllerConfigurationSpec`     {#ControllerConfigurationSpec}
    

**Appears in:**

- [ControllerManager](#ControllerManager)


<p>ControllerConfigurationSpec 定义注册到 manager 的控制器的全局配置。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>groupKindConcurrency</code><br/>
<code>map[string]int</code>
</td>
<td>
   <p>GroupKindConcurrency 是从 Kind 到控制器并发重调度的映射。</p>
<p>当在 manager 中使用 builder 工具注册控制器时，用户必须在 For(...) 调用中指定控制器重调度的类型。
如果对象的种类与该映射中的某个键匹配，则该控制器的并发设置为指定的数量。</p>
<p>键的格式应与 GroupKind.String() 一致，例如 ReplicaSet 在 apps 组（无论版本如何）中将是 <code>ReplicaSet.apps</code>。</p>
</td>
</tr>
<tr><td><code>cacheSyncTimeout</code><br/>
<a href="https://pkg.go.dev/time#Duration"><code>time.Duration</code></a>
</td>
<td>
   <p>CacheSyncTimeout 是指设置缓存同步超时的时间限制。
如果未设置，则默认为 2 分钟。</p>
</td>
</tr>
</tbody>
</table>

## `ControllerHealth`     {#ControllerHealth}
    

**Appears in:**

- [ControllerManager](#ControllerManager)


<p>ControllerHealth 定义健康检查配置。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>healthProbeBindAddress</code><br/>
<code>string</code>
</td>
<td>
   <p>HealthProbeBindAddress 是控制器用于健康探针服务的 TCP 地址。
可以设置为 &quot;0&quot; 或 &quot;&quot; 以禁用健康探针服务。</p>
</td>
</tr>
<tr><td><code>readinessEndpointName</code><br/>
<code>string</code>
</td>
<td>
   <p>ReadinessEndpointName，默认为 &quot;readyz&quot;</p>
</td>
</tr>
<tr><td><code>livenessEndpointName</code><br/>
<code>string</code>
</td>
<td>
   <p>LivenessEndpointName，默认为 &quot;healthz&quot;</p>
</td>
</tr>
</tbody>
</table>

## `ControllerManager`     {#ControllerManager}
    

**Appears in:**




<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>webhook</code><br/>
<a href="#ControllerWebhook"><code>ControllerWebhook</code></a>
</td>
<td>
   <p>Webhook 包含控制器 webhook 的配置</p>
</td>
</tr>
<tr><td><code>leaderElection</code><br/>
<a href="https://pkg.go.dev/k8s.io/component-base/config/v1alpha1#LeaderElectionConfiguration"><code>k8s.io/component-base/config/v1alpha1.LeaderElectionConfiguration</code></a>
</td>
<td>
   <p>LeaderElection 是配置 manager.Manager 选举的 LeaderElection 配置</p>
</td>
</tr>
<tr><td><code>metrics</code><br/>
<a href="#ControllerMetrics"><code>ControllerMetrics</code></a>
</td>
<td>
   <p>Metrics 包含控制器指标的配置</p>
</td>
</tr>
<tr><td><code>health</code><br/>
<a href="#ControllerHealth"><code>ControllerHealth</code></a>
</td>
<td>
   <p>Health 包含控制器健康检查的配置</p>
</td>
</tr>
<tr><td><code>pprofBindAddress</code><br/>
<code>string</code>
</td>
<td>
   <p>PprofBindAddress 是控制器用于绑定 pprof 服务的 TCP 地址。
可以设置为 &quot;&quot; 或 &quot;0&quot; 以禁用 pprof 服务。
由于 pprof 可能包含敏感信息，公开前请确保保护好。</p>
</td>
</tr>
<tr><td><code>controller</code><br/>
<a href="#ControllerConfigurationSpec"><code>ControllerConfigurationSpec</code></a>
</td>
<td>
   <p>Controller 包含在此 manager 注册的控制器的全局配置选项。</p>
</td>
</tr>
</tbody>
</table>

## `ControllerMetrics`     {#ControllerMetrics}
    

**Appears in:**

- [ControllerManager](#ControllerManager)


<p>ControllerMetrics 定义指标配置。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>bindAddress</code><br/>
<code>string</code>
</td>
<td>
   <p>BindAddress 是控制器用于绑定 prometheus 指标服务的 TCP 地址。
可以设置为 &quot;0&quot; 以禁用指标服务。</p>
</td>
</tr>
<tr><td><code>enableClusterQueueResources</code><br/>
<code>bool</code>
</td>
<td>
   <p>EnableClusterQueueResources 如果为 true，则会报告集群队列资源使用和配额指标。</p>
</td>
</tr>
</tbody>
</table>

## `ControllerWebhook`     {#ControllerWebhook}
    

**Appears in:**

- [ControllerManager](#ControllerManager)


<p>ControllerWebhook 定义控制器的 webhook 服务器。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>port</code><br/>
<code>int</code>
</td>
<td>
   <p>Port 是 webhook 服务器服务的端口。
用于设置 webhook.Server.Port。</p>
</td>
</tr>
<tr><td><code>host</code><br/>
<code>string</code>
</td>
<td>
   <p>Host 是 webhook 服务器绑定的主机名。
用于设置 webhook.Server.Host。</p>
</td>
</tr>
<tr><td><code>certDir</code><br/>
<code>string</code>
</td>
<td>
   <p>CertDir 是包含服务器密钥和证书的目录。
如果未设置，webhook 服务器会在 {TempDir}/k8s-webhook-server/serving-certs 查找服务器密钥和证书。
服务器密钥和证书必须分别命名为 tls.key 和 tls.crt。</p>
</td>
</tr>
</tbody>
</table>

## `FairSharing`     {#FairSharing}
    

**Appears in:**




<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>enable</code> <B>[Required]</B><br/>
<code>bool</code>
</td>
<td>
   <p>enable 表示是否启用所有 cohort 的公平共享。
默认为 false。</p>
</td>
</tr>
<tr><td><code>preemptionStrategies</code> <B>[Required]</B><br/>
<a href="#PreemptionStrategy"><code>[]PreemptionStrategy</code></a>
</td>
<td>
   <p>preemptionStrategies 表示哪些约束应由抢占满足。
抢占算法仅在抢占者（preemptor）不满足下一个策略时才使用列表中的下一个策略。
可能的值是：</p>
<ul>
<li>LessThanOrEqualToFinalShare：仅当抢占者 CQ 与抢占者工作负载的共享小于或等于被抢占 CQ 的共享时才抢占工作负载。
这可能会优先抢占较小的工作负载，无论优先级或开始时间如何，以保持 CQ 的共享尽可能高。</li>
<li>LessThanInitialShare：仅当抢占者 CQ 与传入工作负载的共享严格小于被抢占 CQ 的共享时才抢占工作负载。
此策略不依赖于被抢占工作负载的共享使用情况。
结果是，策略选择优先抢占具有最低优先级和最新开始时间的作业。
默认策略是 [&quot;LessThanOrEqualToFinalShare&quot;, &quot;LessThanInitialShare&quot;]。</li>
</ul>
</td>
</tr>
</tbody>
</table>

## `Integrations`     {#Integrations}
    

**Appears in:**




<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>frameworks</code> <B>[Required]</B><br/>
<code>[]string</code>
</td>
<td>
   <p>List of framework names to be enabled.
Possible options:</p>
<ul>
<li>&quot;batch/job&quot;</li>
<li>&quot;pod&quot;</li>
<li>&quot;deployment&quot; (requires enabling pod integration)</li>
<li>&quot;statefulset&quot; (requires enabling pod integration)</li>
<li>&quot;leaderworkerset.x-k8s.io/leaderworkerset&quot; (requires enabling pod integration)</li>
</ul>
</td>
</tr>
<tr><td><code>externalFrameworks</code> <B>[Required]</B><br/>
<code>[]string</code>
</td>
<td>
   <p>List of GroupVersionKinds that are managed for Kueue by external controllers;
the expected format is <code>Kind.version.group.com</code>.</p>
</td>
</tr>
<tr><td><code>podOptions</code> <B>[Required]</B><br/>
<a href="#PodIntegrationOptions"><code>PodIntegrationOptions</code></a>
</td>
<td>
   <p>PodOptions 定义 kueue 控制器对 pod 对象的行为
已废弃：此字段将在 v1beta2 移除，请使用 ManagedJobsNamespaceSelector
(https://kueue.sigs.k8s.io/docs/tasks/run/plain_pods/)
代替。</p>
</td>
</tr>
<tr><td><code>labelKeysToCopy</code> <B>[Required]</B><br/>
<code>[]string</code>
</td>
<td>
   <p>labelKeysToCopy 是从作业复制到工作负载对象的标签键列表。
作业不必具有此列表中的所有标签。如果作业缺少此列表中某个键的标签，则构造的工作负载对象将不包含该标签。
在从可组合作业（pod 组）创建工作负载时，如果多个对象具有此列表中某个键的标签，则这些标签的值必须匹配，
否则工作负载创建将失败。标签仅在创建工作负载时复制，即使底层作业的标签已更改，也不会更新。</p>
</td>
</tr>
</tbody>
</table>

## `InternalCertManagement`     {#InternalCertManagement}
    

**Appears in:**




<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>enable</code> <B>[Required]</B><br/>
<code>bool</code>
</td>
<td>
   <p>Enable 控制是否使用内部证书管理。
启用时，Kueue 使用库来生成和自签名证书。
禁用时，您需要通过第三方证书提供 webhook 和指标端点的证书。
此 secret 挂载到 kueue 控制器管理器 pod 中。webhook 的挂载路径为 /tmp/k8s-webhook-server/serving-certs，
指标端点的预期路径为 <code>/etc/kueue/metrics/certs</code>。密钥和证书分别命名为 tls.key 和 tls.crt。</p>
</td>
</tr>
<tr><td><code>webhookServiceName</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>WebhookServiceName 是作为 DNSName 一部分使用的 Service 名称。
默认为 kueue-webhook-service。</p>
</td>
</tr>
<tr><td><code>webhookSecretName</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>WebhookSecretName 是用于存储 CA 和服务器证书的 Secret 名称。
默认为 kueue-webhook-server-cert。</p>
</td>
</tr>
</tbody>
</table>

## `MultiKueue`     {#MultiKueue}
    

**Appears in:**




<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>gcInterval</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#duration-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Duration</code></a>
</td>
<td>
   <p>GCInterval 定义了两次连续垃圾回收运行之间的时间间隔。
默认为 1min。如果为 0，则禁用垃圾回收。</p>
</td>
</tr>
<tr><td><code>origin</code><br/>
<code>string</code>
</td>
<td>
   <p>Origin 定义了一个标签值，用于跟踪 worker 集群中工作负载的创建者。
这用于 multikueue 在其组件（如垃圾收集器）中识别远程对象，
如果本地对应对象不再存在，则删除它们。</p>
</td>
</tr>
<tr><td><code>workerLostTimeout</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#duration-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Duration</code></a>
</td>
<td>
   <p>WorkerLostTimeout 定义了本地工作负载的 multikueue 准入检查状态在与其预留 worker 集群断开连接后保持 Ready 的时间。</p>
<p>默认为 15 分钟。</p>
</td>
</tr>
</tbody>
</table>

## `ObjectRetentionPolicies`     {#ObjectRetentionPolicies}
    

**Appears in:**



<p>ObjectRetentionPolicies 提供 Kueue 管理对象自动删除的配置选项。nil 值禁用所有自动删除。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>workloads</code><br/>
<a href="#WorkloadRetentionPolicy"><code>WorkloadRetentionPolicy</code></a>
</td>
<td>
   <p>Workloads 配置 Workloads 的保留。
一个 nil 值禁用 Workloads 的自动删除。</p>
</td>
</tr>
</tbody>
</table>

## `PodIntegrationOptions`     {#PodIntegrationOptions}
    

**Appears in:**

- [Integrations](#Integrations)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>namespaceSelector</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#labelselector-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector</code></a>
</td>
<td>
   <p>NamespaceSelector 可用于排除一些命名空间从 pod 重调度的范围。</p>
</td>
</tr>
<tr><td><code>podSelector</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#labelselector-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector</code></a>
</td>
<td>
   <p>PodSelector 可用于选择要重调度的 pod。</p>
</td>
</tr>
</tbody>
</table>

## `PreemptionStrategy`     {#PreemptionStrategy}
    
(Alias of `string`)

**Appears in:**

- [FairSharing](#FairSharing)





## `QueueVisibility`     {#QueueVisibility}
    

**Appears in:**




<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>clusterQueues</code> <B>[Required]</B><br/>
<a href="#ClusterQueueVisibility"><code>ClusterQueueVisibility</code></a>
</td>
<td>
   <p>ClusterQueues 是配置，用于暴露集群队列中顶部待处理工作负载的信息。</p>
</td>
</tr>
<tr><td><code>updateIntervalSeconds</code> <B>[Required]</B><br/>
<code>int32</code>
</td>
<td>
   <p>UpdateIntervalSeconds 指定更新队列中顶部待处理工作负载结构的时间间隔。
最小值为 1。
默认为 5。</p>
</td>
</tr>
</tbody>
</table>

## `RequeuingStrategy`     {#RequeuingStrategy}
    

**Appears in:**

- [WaitForPodsReady](#WaitForPodsReady)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>timestamp</code><br/>
<a href="#RequeuingTimestamp"><code>RequeuingTimestamp</code></a>
</td>
<td>
   <p>Timestamp 定义了重新排队 Workload 的时间戳。
可能的值是：</p>
<ul>
<li><code>Eviction</code>（默认）表示从 Workload <code>Evicted</code> 条件获取时间，原因 <code>PodsReadyTimeout</code>。</li>
<li><code>Creation</code> 表示从 Workload .metadata.creationTimestamp 获取时间。</li>
</ul>
</td>
</tr>
<tr><td><code>backoffLimitCount</code><br/>
<code>int32</code>
</td>
<td>
   <p>BackoffLimitCount 定义了最大重试次数。
达到此数量后，工作负载将被停用（<code>.spec.activate</code>=<code>false</code>）。
当为 null 时，工作负载将不断重复重试。</p>
<p>每次回退持续时间约为 &quot;b*2^(n-1)+Rand&quot;，其中：</p>
<ul>
<li>&quot;b&quot; 表示由 &quot;BackoffBaseSeconds&quot; 参数设置的基础，</li>
<li>&quot;n&quot; 表示 &quot;workloadStatus.requeueState.count&quot;，</li>
<li>&quot;Rand&quot; 表示随机抖动。
在此期间，工作负载被视为不合格，
其他工作负载将有机会被接收。
默认情况下，连续重试延迟约为：(60s, 120s, 240s, ...)。</li>
</ul>
<p>默认为 null。</p>
</td>
</tr>
<tr><td><code>backoffBaseSeconds</code><br/>
<code>int32</code>
</td>
<td>
   <p>BackoffBaseSeconds 定义了重新排队已驱逐工作负载的指数回退基础。</p>
<p>默认为 60。</p>
</td>
</tr>
<tr><td><code>backoffMaxSeconds</code><br/>
<code>int32</code>
</td>
<td>
   <p>BackoffMaxSeconds 定义了重新排队已驱逐工作负载的最大回退时间。</p>
<p>默认为 3600。</p>
</td>
</tr>
</tbody>
</table>

## `RequeuingTimestamp`     {#RequeuingTimestamp}
    
(Alias of `string`)

**Appears in:**

- [RequeuingStrategy](#RequeuingStrategy)





## `ResourceTransformation`     {#ResourceTransformation}
    

**Appears in:**

- [Resources](#Resources)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>input</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcename-v1-core"><code>k8s.io/api/core/v1.ResourceName</code></a>
</td>
<td>
   <p>Input 是输入资源的名称。</p>
</td>
</tr>
<tr><td><code>strategy</code> <B>[Required]</B><br/>
<a href="#ResourceTransformationStrategy"><code>ResourceTransformationStrategy</code></a>
</td>
<td>
   <p>Strategy 指定输入资源是否应替换或保留。
默认为 Retain</p>
</td>
</tr>
<tr><td><code>outputs</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcelist-v1-core"><code>k8s.io/api/core/v1.ResourceList</code></a>
</td>
<td>
   <p>Outputs 指定输入资源每单位数量的输出资源和数量。
空 Outputs 与 <code>Replace</code> 策略结合使用会导致 Kueue 忽略输入资源。</p>
</td>
</tr>
</tbody>
</table>

## `ResourceTransformationStrategy`     {#ResourceTransformationStrategy}
    
(Alias of `string`)

**Appears in:**

- [ResourceTransformation](#ResourceTransformation)





## `Resources`     {#Resources}
    

**Appears in:**




<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>excludeResourcePrefixes</code> <B>[Required]</B><br/>
<code>[]string</code>
</td>
<td>
   <p>ExcludedResourcePrefixes 定义哪些资源应被 Kueue 忽略。</p>
</td>
</tr>
<tr><td><code>transformations</code> <B>[Required]</B><br/>
<a href="#ResourceTransformation"><code>[]ResourceTransformation</code></a>
</td>
<td>
   <p>Transformations 定义如何将 PodSpec 资源转换为 Workload 资源请求。
这旨在是一个键为 Input 的映射（由验证代码强制执行）</p>
</td>
</tr>
</tbody>
</table>

## `WaitForPodsReady`     {#WaitForPodsReady}
    

**Appears in:**



<p>WaitForPodsReady 是为作业提供基于时间的全有或全无调度语义的配置，
通过确保所有 pod 在指定时间内就绪（运行并通过就绪探针）。如果超时，则驱逐该工作负载。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>enable</code> <B>[Required]</B><br/>
<code>bool</code>
</td>
<td>
   <p>Enable 表示是否启用等待 Pods 就绪功能。
默认为 false。</p>
</td>
</tr>
<tr><td><code>timeout</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#duration-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Duration</code></a>
</td>
<td>
   <p>Timeout 定义了已接收工作负载达到 PodsReady=true 条件的时间。
当超时时，工作负载被驱逐并重新排队到同一个集群队列。
默认为 5min。</p>
</td>
</tr>
<tr><td><code>blockAdmission</code> <B>[Required]</B><br/>
<code>bool</code>
</td>
<td>
   <p>BlockAdmission 当为 true 时，集群队列将阻止所有后续作业的准入，直到作业达到 PodsReady=true 条件。
此设置仅在 <code>Enable</code> 设置为 true 时才生效。</p>
</td>
</tr>
<tr><td><code>requeuingStrategy</code><br/>
<a href="#RequeuingStrategy"><code>RequeuingStrategy</code></a>
</td>
<td>
   <p>RequeuingStrategy 定义了重新排队 Workload 的策略。</p>
</td>
</tr>
<tr><td><code>recoveryTimeout</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#duration-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Duration</code></a>
</td>
<td>
   <p>RecoveryTimeout 定义了一个可选的超时，从 Workload 被 Admitted 并运行后的最后一次 PodsReady=false 状态开始测量。
这种状态可能发生在 Pod 失败且替换 Pod 等待调度时。
超时后，相应的作业将被暂停，并在回退延迟后重新排队。超时仅在 waitForPodsReady.enable=true 时强制执行。
如果未设置，则没有超时。</p>
</td>
</tr>
</tbody>
</table>

## `WorkloadRetentionPolicy`     {#WorkloadRetentionPolicy}
    

**Appears in:**

- [ObjectRetentionPolicies](#ObjectRetentionPolicies)


<p>WorkloadRetentionPolicy 定义 Workloads 应在何时删除的策略。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>afterFinished</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#duration-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Duration</code></a>
</td>
<td>
   <p>AfterFinished 是 Workload 完成后等待删除的时间。
持续时间为 0 将立即删除。
一个 nil 值禁用自动删除。
使用 metav1.Duration（例如 &quot;10m&quot;, &quot;1h30m&quot;）表示。</p>
</td>
</tr>
<tr><td><code>afterDeactivatedByKueue</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#duration-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Duration</code></a>
</td>
<td>
   <p>AfterDeactivatedByKueue 是 <em>任何</em> Kueue 管理的 Workload（例如 Job、JobSet 或其他自定义工作负载类型）
被 Kueue 标记为 deactivated 后自动删除的时间。
已停用工作负载的删除可能会级联到不是由 Kueue 创建的对象，
因为删除父 Workload 所有者（例如 JobSet）可以触发依赖资源的垃圾回收。
持续时间为 0 将立即删除。
一个 nil 值禁用自动删除。
使用 metav1.Duration（例如 &quot;10m&quot;, &quot;1h30m&quot;）表示。</p>
</td>
</tr>
</tbody>
</table>