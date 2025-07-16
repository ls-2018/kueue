---
title: Kueue API
content_type: tool-reference
package: kueue.x-k8s.io/v1beta1
auto_generated: true
description: Generated API reference documentation for kueue.x-k8s.io/v1beta1.
---


## Resource Types 


- [AdmissionCheck](#kueue-x-k8s-io-v1beta1-AdmissionCheck)
- [ClusterQueue](#kueue-x-k8s-io-v1beta1-ClusterQueue)
- [Cohort](#kueue-x-k8s-io-v1beta1-Cohort)
- [LocalQueue](#kueue-x-k8s-io-v1beta1-LocalQueue)
- [MultiKueueCluster](#kueue-x-k8s-io-v1beta1-MultiKueueCluster)
- [MultiKueueConfig](#kueue-x-k8s-io-v1beta1-MultiKueueConfig)
- [ProvisioningRequestConfig](#kueue-x-k8s-io-v1beta1-ProvisioningRequestConfig)
- [ResourceFlavor](#kueue-x-k8s-io-v1beta1-ResourceFlavor)
- [Workload](#kueue-x-k8s-io-v1beta1-Workload)
- [WorkloadPriorityClass](#kueue-x-k8s-io-v1beta1-WorkloadPriorityClass)
  

## `AdmissionCheck`     {#kueue-x-k8s-io-v1beta1-AdmissionCheck}
    

**Appears in:**



<p>AdmissionCheck 是 admissionchecks API 的 Schema</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
<tr><td><code>apiVersion</code><br/>string</td><td><code>kueue.x-k8s.io/v1beta1</code></td></tr>
<tr><td><code>kind</code><br/>string</td><td><code>AdmissionCheck</code></td></tr>
    
  
<tr><td><code>spec</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionCheckSpec"><code>AdmissionCheckSpec</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
<tr><td><code>status</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionCheckStatus"><code>AdmissionCheckStatus</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `ClusterQueue`     {#kueue-x-k8s-io-v1beta1-ClusterQueue}
    

**Appears in:**



<p>ClusterQueue is the Schema for the clusterQueue API.</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
<tr><td><code>apiVersion</code><br/>string</td><td><code>kueue.x-k8s.io/v1beta1</code></td></tr>
<tr><td><code>kind</code><br/>string</td><td><code>ClusterQueue</code></td></tr>
    
  
<tr><td><code>spec</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ClusterQueueSpec"><code>ClusterQueueSpec</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
<tr><td><code>status</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ClusterQueueStatus"><code>ClusterQueueStatus</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `Cohort`     {#kueue-x-k8s-io-v1beta1-Cohort}
    

**Appears in:**



<p>Cohort 定义了 Cohorts API。</p>
<p>分层 Cohort（有父级的 Cohort）自 v0.11 起兼容公平共享。在 v0.9 和 v0.10 中同时使用这些特性是不支持的，会导致未定义行为。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
<tr><td><code>apiVersion</code><br/>string</td><td><code>kueue.x-k8s.io/v1beta1</code></td></tr>
<tr><td><code>kind</code><br/>string</td><td><code>Cohort</code></td></tr>
    
  
<tr><td><code>spec</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-CohortSpec"><code>CohortSpec</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
<tr><td><code>status</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-CohortStatus"><code>CohortStatus</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `LocalQueue`     {#kueue-x-k8s-io-v1beta1-LocalQueue}
    

**Appears in:**



<p>LocalQueue is the Schema for the localQueues API
LocalQueue 是 localQueues API 的架构</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
<tr><td><code>apiVersion</code><br/>string</td><td><code>kueue.x-k8s.io/v1beta1</code></td></tr>
<tr><td><code>kind</code><br/>string</td><td><code>LocalQueue</code></td></tr>
    
  
<tr><td><code>spec</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-LocalQueueSpec"><code>LocalQueueSpec</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
<tr><td><code>status</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-LocalQueueStatus"><code>LocalQueueStatus</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `MultiKueueCluster`     {#kueue-x-k8s-io-v1beta1-MultiKueueCluster}
    

**Appears in:**



<p>MultiKueueCluster 是 multikueue API 的 Schema。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
<tr><td><code>apiVersion</code><br/>string</td><td><code>kueue.x-k8s.io/v1beta1</code></td></tr>
<tr><td><code>kind</code><br/>string</td><td><code>MultiKueueCluster</code></td></tr>
    
  
<tr><td><code>spec</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-MultiKueueClusterSpec"><code>MultiKueueClusterSpec</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
<tr><td><code>status</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-MultiKueueClusterStatus"><code>MultiKueueClusterStatus</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `MultiKueueConfig`     {#kueue-x-k8s-io-v1beta1-MultiKueueConfig}
    

**Appears in:**



<p>MultiKueueConfig 是 multikueue API 的 Schema。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
<tr><td><code>apiVersion</code><br/>string</td><td><code>kueue.x-k8s.io/v1beta1</code></td></tr>
<tr><td><code>kind</code><br/>string</td><td><code>MultiKueueConfig</code></td></tr>
    
  
<tr><td><code>spec</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-MultiKueueConfigSpec"><code>MultiKueueConfigSpec</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `ProvisioningRequestConfig`     {#kueue-x-k8s-io-v1beta1-ProvisioningRequestConfig}
    

**Appears in:**



<p>ProvisioningRequestConfig 是 provisioningrequestconfig API 的 Schema</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
<tr><td><code>apiVersion</code><br/>string</td><td><code>kueue.x-k8s.io/v1beta1</code></td></tr>
<tr><td><code>kind</code><br/>string</td><td><code>ProvisioningRequestConfig</code></td></tr>
    
  
<tr><td><code>spec</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ProvisioningRequestConfigSpec"><code>ProvisioningRequestConfigSpec</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `ResourceFlavor`     {#kueue-x-k8s-io-v1beta1-ResourceFlavor}
    

**Appears in:**



<p>ResourceFlavor is the Schema for the resourceflavors API.
ResourceFlavor 是 resourceflavors API 的架构。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
<tr><td><code>apiVersion</code><br/>string</td><td><code>kueue.x-k8s.io/v1beta1</code></td></tr>
<tr><td><code>kind</code><br/>string</td><td><code>ResourceFlavor</code></td></tr>
    
  
<tr><td><code>spec</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ResourceFlavorSpec"><code>ResourceFlavorSpec</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `Workload`     {#kueue-x-k8s-io-v1beta1-Workload}
    

**Appears in:**



<p>Workload 是工作负载 API 的 Schema</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
<tr><td><code>apiVersion</code><br/>string</td><td><code>kueue.x-k8s.io/v1beta1</code></td></tr>
<tr><td><code>kind</code><br/>string</td><td><code>Workload</code></td></tr>
    
  
<tr><td><code>spec</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-WorkloadSpec"><code>WorkloadSpec</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
<tr><td><code>status</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-WorkloadStatus"><code>WorkloadStatus</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `WorkloadPriorityClass`     {#kueue-x-k8s-io-v1beta1-WorkloadPriorityClass}
    

**Appears in:**



<p>WorkloadPriorityClass is the Schema for the workloadPriorityClass API
WorkloadPriorityClass 是 workloadPriorityClass API 的模式定义</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
<tr><td><code>apiVersion</code><br/>string</td><td><code>kueue.x-k8s.io/v1beta1</code></td></tr>
<tr><td><code>kind</code><br/>string</td><td><code>WorkloadPriorityClass</code></td></tr>
    
  
<tr><td><code>value</code> <B>[Required]</B><br/>
<code>int32</code>
</td>
<td>
   <p>value represents the integer value of this workloadPriorityClass. This is the actual priority that workloads
receive when jobs have the name of this class in their workloadPriorityClass label.
Changing the value of workloadPriorityClass doesn't affect the priority of workloads that were already created.
value 表示此 workloadPriorityClass 的整数值。这是当作业在其 workloadPriorityClass 标签中具有此类名称时，工作负载实际获得的优先级。
更改 workloadPriorityClass 的 value 不会影响已创建工作负载的优先级。</p>
</td>
</tr>
<tr><td><code>description</code><br/>
<code>string</code>
</td>
<td>
   <p>description is an arbitrary string that usually provides guidelines on
when this workloadPriorityClass should be used.
description 是一个任意字符串，通常用于提供何时应使用此 workloadPriorityClass 的指导。</p>
</td>
</tr>
</tbody>
</table>

## `Admission`     {#kueue-x-k8s-io-v1beta1-Admission}
    

**Appears in:**

- [WorkloadStatus](#kueue-x-k8s-io-v1beta1-WorkloadStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>clusterQueue</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ClusterQueueReference"><code>ClusterQueueReference</code></a>
</td>
<td>
   <p>clusterQueue 是集群队列的名称，该集群队列接纳了此 workload。</p>
</td>
</tr>
<tr><td><code>podSetAssignments</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-PodSetAssignment"><code>[]PodSetAssignment</code></a>
</td>
<td>
   <p>PodSetAssignments 持有 .spec.podSets 条目对每个的接纳结果。</p>
</td>
</tr>
</tbody>
</table>

## `AdmissionCheckParametersReference`     {#kueue-x-k8s-io-v1beta1-AdmissionCheckParametersReference}
    

**Appears in:**

- [AdmissionCheckSpec](#kueue-x-k8s-io-v1beta1-AdmissionCheckSpec)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>apiGroup</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>ApiGroup 是被引用资源的组。</p>
</td>
</tr>
<tr><td><code>kind</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>Kind 是被引用资源的类型。</p>
</td>
</tr>
<tr><td><code>name</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>Name 是被引用资源的名称。</p>
</td>
</tr>
</tbody>
</table>

## `AdmissionCheckReference`     {#kueue-x-k8s-io-v1beta1-AdmissionCheckReference}
    
(Alias of `string`)

**Appears in:**

- [AdmissionCheckState](#kueue-x-k8s-io-v1beta1-AdmissionCheckState)

- [AdmissionCheckStrategyRule](#kueue-x-k8s-io-v1beta1-AdmissionCheckStrategyRule)

- [ClusterQueueSpec](#kueue-x-k8s-io-v1beta1-ClusterQueueSpec)


<p>AdmissionCheckReference 是接纳检查的名称。</p>




## `AdmissionCheckSpec`     {#kueue-x-k8s-io-v1beta1-AdmissionCheckSpec}
    

**Appears in:**

- [AdmissionCheck](#kueue-x-k8s-io-v1beta1-AdmissionCheck)


<p>AdmissionCheckSpec 定义 AdmissionCheck 的期望状态</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>controllerName</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>controllerName 标识处理 AdmissionCheck 的控制器，不一定是 Kubernetes Pod 或 Deployment 名称。不能为空。</p>
</td>
</tr>
<tr><td><code>retryDelayMinutes</code><br/>
<code>int64</code>
</td>
<td>
   <p>RetryDelayMinutes 指定检查失败（转为 False）后，工作负载保持挂起的时间。延迟期过后，检查状态变为 &quot;Unknown&quot;。默认 15 分钟。
已废弃：retryDelayMinutes 自 v0.8 起已废弃，将在 v1beta2 移除。</p>
</td>
</tr>
<tr><td><code>parameters</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionCheckParametersReference"><code>AdmissionCheckParametersReference</code></a>
</td>
<td>
   <p>Parameters 标识带有附加参数的检查配置。</p>
</td>
</tr>
</tbody>
</table>

## `AdmissionCheckState`     {#kueue-x-k8s-io-v1beta1-AdmissionCheckState}
    

**Appears in:**

- [WorkloadStatus](#kueue-x-k8s-io-v1beta1-WorkloadStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionCheckReference"><code>AdmissionCheckReference</code></a>
</td>
<td>
   <p>name 标识接纳检查。</p>
</td>
</tr>
<tr><td><code>state</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-CheckState"><code>CheckState</code></a>
</td>
<td>
   <p>admissionCheck的状态，Pending, Ready, Retry, Rejected</p>
</td>
</tr>
<tr><td><code>lastTransitionTime</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Time</code></a>
</td>
<td>
   <p>lastTransitionTime 是条件从一种状态转换到另一种状态的最后时间。
这应该是底层条件发生变化的时间。 如果不知道，则使用 API 字段更改的时间是可接受的。</p>
</td>
</tr>
<tr><td><code>message</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>message 是人类可读的消息，指示转换的详细信息。
这可能是空字符串。</p>
</td>
</tr>
<tr><td><code>podSetUpdates</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-PodSetUpdate"><code>[]PodSetUpdate</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `AdmissionCheckStatus`     {#kueue-x-k8s-io-v1beta1-AdmissionCheckStatus}
    

**Appears in:**

- [AdmissionCheck](#kueue-x-k8s-io-v1beta1-AdmissionCheck)


<p>AdmissionCheckStatus 定义 AdmissionCheck 的观测状态</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>conditions</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta"><code>[]k8s.io/apimachinery/pkg/apis/meta/v1.Condition</code></a>
</td>
<td>
   <p>conditions 保存 AdmissionCheck 当前状态的最新可用观测信息。</p>
</td>
</tr>
</tbody>
</table>

## `AdmissionCheckStrategyRule`     {#kueue-x-k8s-io-v1beta1-AdmissionCheckStrategyRule}
    

**Appears in:**

- [AdmissionChecksStrategy](#kueue-x-k8s-io-v1beta1-AdmissionChecksStrategy)


<p>AdmissionCheckStrategyRule defines rules for a single AdmissionCheck
AdmissionCheckStrategyRule 定义单个 AdmissionCheck 的规则</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionCheckReference"><code>AdmissionCheckReference</code></a>
</td>
<td>
   <p>name is an AdmissionCheck's name.
name 是 AdmissionCheck 的名称。</p>
</td>
</tr>
<tr><td><code>onFlavors</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-ResourceFlavorReference"><code>[]ResourceFlavorReference</code></a>
</td>
<td>
   <p>onFlavors is a list of ResourceFlavors' names that this AdmissionCheck should run for.
onFlavors 是此 AdmissionCheck 应运行的 ResourceFlavors 名称列表。
If empty, the AdmissionCheck will run for all workloads submitted to the ClusterQueue.
如果为空，则 AdmissionCheck 将对提交到 ClusterQueue 的所有工作负载运行。</p>
</td>
</tr>
</tbody>
</table>

## `AdmissionChecksStrategy`     {#kueue-x-k8s-io-v1beta1-AdmissionChecksStrategy}
    

**Appears in:**

- [ClusterQueueSpec](#kueue-x-k8s-io-v1beta1-ClusterQueueSpec)


<p>AdmissionChecksStrategy defines a strategy for a AdmissionCheck.
AdmissionChecksStrategy 定义 AdmissionCheck 的策略。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>admissionChecks</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionCheckStrategyRule"><code>[]AdmissionCheckStrategyRule</code></a>
</td>
<td>
   <p>admissionChecks is a list of strategies for AdmissionChecks
admissionChecks 是 AdmissionChecks 的策略列表</p>
</td>
</tr>
</tbody>
</table>

## `AdmissionFairSharingStatus`     {#kueue-x-k8s-io-v1beta1-AdmissionFairSharingStatus}
    

**Appears in:**

- [FairSharingStatus](#kueue-x-k8s-io-v1beta1-FairSharingStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>consumedResources</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcelist-v1-core"><code>k8s.io/api/core/v1.ResourceList</code></a>
</td>
<td>
   <p>ConsumedResources 表示资源随时间的聚合使用量，应用了衰减函数。
如果在 Kueue 配置中启用了使用量消耗功能，则会填充值。</p>
</td>
</tr>
<tr><td><code>lastUpdate</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Time</code></a>
</td>
<td>
   <p>LastUpdate 是份额和已消耗资源被更新的时间。</p>
</td>
</tr>
</tbody>
</table>

## `AdmissionMode`     {#kueue-x-k8s-io-v1beta1-AdmissionMode}
    
(Alias of `string`)

**Appears in:**

- [AdmissionScope](#kueue-x-k8s-io-v1beta1-AdmissionScope)





## `AdmissionScope`     {#kueue-x-k8s-io-v1beta1-AdmissionScope}
    

**Appears in:**

- [ClusterQueueSpec](#kueue-x-k8s-io-v1beta1-ClusterQueueSpec)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>admissionMode</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionMode"><code>AdmissionMode</code></a>
</td>
<td>
   <p>AdmissionMode 表示 AdmissionScope 中 AdmissionFairSharing 应使用的模式。
可能的值有：</p>
<ul>
<li>UsageBasedAdmissionFairSharing</li>
<li>NoAdmissionFairSharing</li>
</ul>
</td>
</tr>
</tbody>
</table>

## `BorrowWithinCohort`     {#kueue-x-k8s-io-v1beta1-BorrowWithinCohort}
    

**Appears in:**

- [ClusterQueuePreemption](#kueue-x-k8s-io-v1beta1-ClusterQueuePreemption)


<p>BorrowWithinCohort 包含允许在借用时抢占 cohort 内工作负载的配置。仅适用于经典抢占，不适用于公平共享。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>policy</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-BorrowWithinCohortPolicy"><code>BorrowWithinCohortPolicy</code></a>
</td>
<td>
   <p>policy 决定借用时在 cohort 内回收配额的抢占策略。
可选值：</p>
<ul>
<li><code>Never</code>（默认）：不允许借用工作负载在 cohort 内其他 ClusterQueue 抢占。</li>
<li><code>LowerPriority</code>：允许借用工作负载在 cohort 内其他 ClusterQueue 抢占，但仅限于被抢占工作负载优先级较低时。</li>
</ul>
</td>
</tr>
<tr><td><code>maxPriorityThreshold</code><br/>
<code>int32</code>
</td>
<td>
   <p>maxPriorityThreshold 允许将可被借用工作负载抢占的工作负载范围限制为优先级小于等于指定阈值的工作负载。
如果未指定阈值，则任何满足策略的工作负载都可以被借用工作负载抢占。</p>
</td>
</tr>
</tbody>
</table>

## `BorrowWithinCohortPolicy`     {#kueue-x-k8s-io-v1beta1-BorrowWithinCohortPolicy}
    
(Alias of `string`)

**Appears in:**

- [BorrowWithinCohort](#kueue-x-k8s-io-v1beta1-BorrowWithinCohort)


<p>BorrowWithinCohortPolicy defines the policy for borrowing within cohort.
BorrowWithinCohortPolicy 定义 cohort 内借用时的策略。</p>




## `CheckState`     {#kueue-x-k8s-io-v1beta1-CheckState}
    
(Alias of `string`)

**Appears in:**

- [AdmissionCheckState](#kueue-x-k8s-io-v1beta1-AdmissionCheckState)





## `ClusterQueuePendingWorkload`     {#kueue-x-k8s-io-v1beta1-ClusterQueuePendingWorkload}
    

**Appears in:**

- [ClusterQueuePendingWorkloadsStatus](#kueue-x-k8s-io-v1beta1-ClusterQueuePendingWorkloadsStatus)


<p>ClusterQueuePendingWorkload contains the information identifying a pending workload
in the cluster queue.
ClusterQueuePendingWorkload 包含标识 cluster queue 中待处理工作负载的信息。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>Name 表示待处理工作负载的名称。</p>
</td>
</tr>
<tr><td><code>namespace</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>Namespace 表示待处理工作负载的命名空间。</p>
</td>
</tr>
</tbody>
</table>

## `ClusterQueuePendingWorkloadsStatus`     {#kueue-x-k8s-io-v1beta1-ClusterQueuePendingWorkloadsStatus}
    

**Appears in:**

- [ClusterQueueStatus](#kueue-x-k8s-io-v1beta1-ClusterQueueStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>clusterQueuePendingWorkload</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-ClusterQueuePendingWorkload"><code>[]ClusterQueuePendingWorkload</code></a>
</td>
<td>
   <p>Head 包含最前面的待处理工作负载列表。</p>
</td>
</tr>
<tr><td><code>lastChangeTime</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Time</code></a>
</td>
<td>
   <p>LastChangeTime 表示结构最后一次变更的时间。</p>
</td>
</tr>
</tbody>
</table>

## `ClusterQueuePreemption`     {#kueue-x-k8s-io-v1beta1-ClusterQueuePreemption}
    

**Appears in:**

- [ClusterQueueSpec](#kueue-x-k8s-io-v1beta1-ClusterQueueSpec)


<p>ClusterQueuePreemption 包含从本 ClusterQueue 或其 cohort 抢占工作负载的策略。</p>
<p>抢占可配置在以下场景下工作：</p>
<ul>
<li>当工作负载适合 ClusterQueue 的 nominal quota，但该配额当前被 cohort 中其他 ClusterQueue 借用时。我们会抢占其他 ClusterQueue 的工作负载，以允许本 ClusterQueue 回收其 nominal quota。通过 reclaimWithinCohort 配置。</li>
<li>当工作负载不适合 ClusterQueue 的 nominal quota，且 ClusterQueue 中有更低优先级的已接收工作负载时。通过 withinClusterQueue 配置。</li>
<li>当工作负载可以通过借用和抢占 cohort 中低优先级工作负载来适配时。通过 borrowWithinCohort 配置。</li>
<li>启用 FairSharing 时，为保持未用资源的公平分配。详见 FairSharing 文档。</li>
</ul>
<p>抢占算法会尝试找到最小集合的工作负载进行抢占，以容纳待处理工作负载，优先抢占优先级较低的工作负载。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>reclaimWithinCohort</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-PreemptionPolicy"><code>PreemptionPolicy</code></a>
</td>
<td>
   <p>reclaimWithinCohort 决定待处理工作负载是否可以抢占 cohort 中超出 nominal quota 的其他 ClusterQueue 的工作负载。可选值：</p>
<ul>
<li><code>Never</code>（默认）：不抢占 cohort 中的工作负载。</li>
<li><code>LowerPriority</code>：<strong>经典抢占</strong>：如果待处理工作负载适合其 ClusterQueue 的 nominal quota，仅抢占 cohort 中优先级较低的工作负载。<strong>公平共享</strong>：仅抢占 cohort 中优先级较低且满足公平共享抢占策略的工作负载。</li>
<li><code>Any</code>：<strong>经典抢占</strong>：如果待处理工作负载适合其 ClusterQueue 的 nominal quota，可抢占 cohort 中任何工作负载，无论优先级。<strong>公平共享</strong>：抢占 cohort 中满足公平共享抢占策略的工作负载。</li>
</ul>
</td>
</tr>
<tr><td><code>borrowWithinCohort</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-BorrowWithinCohort"><code>BorrowWithinCohort</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
<tr><td><code>withinClusterQueue</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-PreemptionPolicy"><code>PreemptionPolicy</code></a>
</td>
<td>
   <p>withinClusterQueue 决定不适合其 ClusterQueue nominal quota 的待处理工作负载是否可以抢占 ClusterQueue 中的活动工作负载。可选值：</p>
<ul>
<li><code>Never</code>（默认）：不抢占 ClusterQueue 中的工作负载。</li>
<li><code>LowerPriority</code>：仅抢占 ClusterQueue 中优先级低于待处理工作负载的工作负载。</li>
<li><code>LowerOrNewerEqualPriority</code>：仅抢占 ClusterQueue 中优先级低于待处理工作负载，或优先级相同但比待处理工作负载新的工作负载。</li>
</ul>
</td>
</tr>
</tbody>
</table>

## `ClusterQueueReference`     {#kueue-x-k8s-io-v1beta1-ClusterQueueReference}
    
(Alias of `string`)

**Appears in:**

- [Admission](#kueue-x-k8s-io-v1beta1-Admission)

- [LocalQueueSpec](#kueue-x-k8s-io-v1beta1-LocalQueueSpec)


<p>ClusterQueueReference 是 ClusterQueue 的名称。
必须为 DNS（RFC 1123）格式，最大长度为 253 个字符。</p>




## `ClusterQueueSpec`     {#kueue-x-k8s-io-v1beta1-ClusterQueueSpec}
    

**Appears in:**

- [ClusterQueue](#kueue-x-k8s-io-v1beta1-ClusterQueue)


<p>ClusterQueueSpec 定义 ClusterQueue 的期望状态</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>resourceGroups</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ResourceGroup"><code>[]ResourceGroup</code></a>
</td>
<td>
   <p>resourceGroups 描述资源组。
每个资源组定义资源列表和为这些资源提供配额的 flavor 列表。
每个资源和每个 flavor 只能属于一个资源组。
resourceGroups 最多可有 16 个。</p>
</td>
</tr>
<tr><td><code>cohort</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-CohortReference"><code>CohortReference</code></a>
</td>
<td>
   <p>cohort 表示该 ClusterQueue 所属的 cohort。属于同一 cohort 的 CQ 可以相互借用未使用的资源。</p>
<p>CQ 只能属于一个借用 cohort。提交到引用该 CQ 的队列的工作负载可以从 cohort 中的任何 CQ 借用配额。
只有 CQ 中列出的 [resource, flavor] 对的配额可以被借用。
如果为空，则该 ClusterQueue 不能从其他 ClusterQueue 借用，也不能被其他 ClusterQueue 借用。</p>
<p>cohort 是将 CQ 关联在一起的名称，但不引用任何对象。</p>
</td>
</tr>
<tr><td><code>queueingStrategy</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-QueueingStrategy"><code>QueueingStrategy</code></a>
</td>
<td>
   <p>QueueingStrategy 表示该 ClusterQueue 下队列中工作负载的排队策略。
当前支持的策略：</p>
<ul>
<li>StrictFIFO：工作负载严格按创建时间排序。无法接收的旧工作负载会阻塞新工作负载的接收，即使新工作负载适合现有配额。</li>
<li>BestEffortFIFO：工作负载按创建时间排序，但无法接收的旧工作负载不会阻塞适合现有配额的新工作负载的接收。</li>
</ul>
</td>
</tr>
<tr><td><code>namespaceSelector</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#labelselector-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector</code></a>
</td>
<td>
   <p>namespaceSelector 定义哪些命名空间可以向该 clusterQueue 提交工作负载。除了基本策略支持外，应使用策略代理（如 Gatekeeper）来实施更高级的策略。
默认为 null，即无选择器（无命名空间有资格）。如果设置为空选择器 <code>{}</code>，则所有命名空间都有资格。</p>
</td>
</tr>
<tr><td><code>flavorFungibility</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-FlavorFungibility"><code>FlavorFungibility</code></a>
</td>
<td>
   <p>flavorFungibility 定义在评估 flavor 时，工作负载是否应在借用或抢占前尝试下一个 flavor。</p>
</td>
</tr>
<tr><td><code>preemption</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ClusterQueuePreemption"><code>ClusterQueuePreemption</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
<tr><td><code>admissionChecks</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionCheckReference"><code>[]AdmissionCheckReference</code></a>
</td>
<td>
   <p>admissionChecks 列出该 ClusterQueue 所需的 AdmissionChecks。不能与 AdmissionCheckStrategy 同时使用。</p>
</td>
</tr>
<tr><td><code>admissionChecksStrategy</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionChecksStrategy"><code>AdmissionChecksStrategy</code></a>
</td>
<td>
   <p>admissionCheckStrategy 定义确定哪些 ResourceFlavors 需要 AdmissionChecks 的策略列表。该属性不能与 'admissionChecks' 属性同时使用。</p>
</td>
</tr>
<tr><td><code>stopPolicy</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-StopPolicy"><code>StopPolicy</code></a>
</td>
<td>
   <p>stopPolicy - 如果设置为非 None，则该 ClusterQueue 被视为非活动状态，不会进行新的配额预留。</p>
<p>根据其值，相关工作负载将：</p>
<ul>
<li>None - 工作负载被接收</li>
<li>HoldAndDrain - 已接收的工作负载被驱逐，预留中的工作负载将取消预留。</li>
<li>Hold - 已接收的工作负载将运行至完成，预留中的工作负载将取消预留。</li>
</ul>
</td>
</tr>
<tr><td><code>fairSharing</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-FairSharing"><code>FairSharing</code></a>
</td>
<td>
   <p>fairSharing 定义该 ClusterQueue 参与公平共享时的属性。仅在 Kueue 配置中启用公平共享时才相关。</p>
</td>
</tr>
<tr><td><code>admissionScope</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionScope"><code>AdmissionScope</code></a>
</td>
<td>
   <p>admissionScope 表示 ClusterQueue 是否使用公平共享资源配额</p>
</td>
</tr>
</tbody>
</table>

## `ClusterQueueStatus`     {#kueue-x-k8s-io-v1beta1-ClusterQueueStatus}
    

**Appears in:**

- [ClusterQueue](#kueue-x-k8s-io-v1beta1-ClusterQueue)


<p>ClusterQueueStatus 定义 ClusterQueue 的观测状态</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>flavorsReservation</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-FlavorUsage"><code>[]FlavorUsage</code></a>
</td>
<td>
   <p>flavorsReservation 是当前被分配到此 ClusterQueue 的工作负载所使用的按 flavor 划分的预留配额。</p>
</td>
</tr>
<tr><td><code>flavorsUsage</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-FlavorUsage"><code>[]FlavorUsage</code></a>
</td>
<td>
   <p>flavorsUsage 是当前被此 ClusterQueue 接收的工作负载所使用的按 flavor 划分的已用配额。</p>
</td>
</tr>
<tr><td><code>pendingWorkloads</code><br/>
<code>int32</code>
</td>
<td>
   <p>pendingWorkloads 是当前等待被此 clusterQueue 接收的工作负载数量。</p>
</td>
</tr>
<tr><td><code>reservingWorkloads</code><br/>
<code>int32</code>
</td>
<td>
   <p>reservingWorkloads 是当前在此 clusterQueue 中预留配额的工作负载数量。</p>
</td>
</tr>
<tr><td><code>admittedWorkloads</code><br/>
<code>int32</code>
</td>
<td>
   <p>admittedWorkloads 是当前已被此 clusterQueue 接收且尚未完成的工作负载数量。</p>
</td>
</tr>
<tr><td><code>conditions</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta"><code>[]k8s.io/apimachinery/pkg/apis/meta/v1.Condition</code></a>
</td>
<td>
   <p>conditions 保存 ClusterQueue 当前状态的最新可用观测信息。</p>
</td>
</tr>
<tr><td><code>pendingWorkloadsStatus</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-ClusterQueuePendingWorkloadsStatus"><code>ClusterQueuePendingWorkloadsStatus</code></a>
</td>
<td>
   <p>PendingWorkloadsStatus 包含关于 cluster queue 中当前待处理工作负载状态的信息。
已弃用：此字段将在 v1beta2 移除，请改用 VisibilityOnDemand。</p>
</td>
</tr>
<tr><td><code>fairSharing</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-FairSharingStatus"><code>FairSharingStatus</code></a>
</td>
<td>
   <p>fairSharing 包含此 ClusterQueue 参与公平共享时的当前状态。
仅在 Kueue 配置中启用公平共享时记录。</p>
</td>
</tr>
</tbody>
</table>

## `CohortReference`     {#kueue-x-k8s-io-v1beta1-CohortReference}
    
(Alias of `string`)

**Appears in:**

- [ClusterQueueSpec](#kueue-x-k8s-io-v1beta1-ClusterQueueSpec)

- [CohortSpec](#kueue-x-k8s-io-v1beta1-CohortSpec)


<p>CohortReference 是 Cohort 的名称。</p>
<p>Cohort 名称的校验等同于对象名称的校验：DNS（RFC 1123）中的子域名。</p>




## `CohortSpec`     {#kueue-x-k8s-io-v1beta1-CohortSpec}
    

**Appears in:**

- [Cohort](#kueue-x-k8s-io-v1beta1-Cohort)


<p>CohortSpec 定义了 Cohort 的期望状态</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>parentName</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-CohortReference"><code>CohortReference</code></a>
</td>
<td>
   <p>ParentName 引用该 Cohort 父级的名称（如果有）。它满足以下三种情况之一：</p>
<ol>
<li>未设置。本 Cohort 是其 Cohort 树的根。</li>
<li>引用了一个不存在的 Cohort。我们使用默认 Cohort（无借用/出借限制）。</li>
<li>引用了一个存在的 Cohort。</li>
</ol>
<p>如果创建了循环，我们会禁用该 Cohort 的所有成员，包括 ClusterQueues，直到循环被移除。在循环存在期间，阻止进一步的准入。</p>
</td>
</tr>
<tr><td><code>resourceGroups</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ResourceGroup"><code>[]ResourceGroup</code></a>
</td>
<td>
   <p>ResourceGroups 描述了资源和风味的分组。每个 ResourceGroup 定义了一组资源和一组为这些资源提供配额的风味。每个资源和每个风味只能属于一个 ResourceGroup。一个 Cohort 最多可以有 16 个 ResourceGroup。</p>
<p>BorrowingLimit 限制该 Cohort 子树成员可以从父子树借用的资源量。</p>
<p>LendingLimit 限制该 Cohort 子树成员可以借给父子树的资源量。</p>
<p>只有当 Cohort 有父级时，才能设置借用和出借限制。否则，webhook 会拒绝 Cohort 的创建/更新。</p>
</td>
</tr>
<tr><td><code>fairSharing</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-FairSharing"><code>FairSharing</code></a>
</td>
<td>
   <p>fairSharing 定义了 Cohort 参与公平共享时的属性。仅当 Kueue 配置中启用公平共享时，这些值才有意义。</p>
</td>
</tr>
</tbody>
</table>

## `CohortStatus`     {#kueue-x-k8s-io-v1beta1-CohortStatus}
    

**Appears in:**

- [Cohort](#kueue-x-k8s-io-v1beta1-Cohort)


<p>CohortStatus 定义了 Cohort 的观测状态。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>fairSharing</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-FairSharingStatus"><code>FairSharingStatus</code></a>
</td>
<td>
   <p>fairSharing 包含该 Cohort 参与公平共享时的当前状态。
仅当 Kueue 配置中启用公平共享时才会记录。</p>
</td>
</tr>
</tbody>
</table>

## `DelayedTopologyRequestState`     {#kueue-x-k8s-io-v1beta1-DelayedTopologyRequestState}
    
(Alias of `string`)

**Appears in:**

- [PodSetAssignment](#kueue-x-k8s-io-v1beta1-PodSetAssignment)


<p>DelayedTopologyRequestState 表示延迟拓扑请求的状态。</p>




## `FairSharing`     {#kueue-x-k8s-io-v1beta1-FairSharing}
    

**Appears in:**

- [ClusterQueueSpec](#kueue-x-k8s-io-v1beta1-ClusterQueueSpec)

- [CohortSpec](#kueue-x-k8s-io-v1beta1-CohortSpec)

- [LocalQueueSpec](#kueue-x-k8s-io-v1beta1-LocalQueueSpec)


<p>FairSharing 包含 ClusterQueue 或 Cohort 参与公平共享时的属性。</p>
<p>公平共享自 v0.11 起兼容分层 Cohort（任何有父级的 Cohort）。在 V0.9 和 V0.10 中同时使用这些特性是不支持的，会导致未定义行为。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>weight</code> <B>[Required]</B><br/>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/api/resource#Quantity"><code>k8s.io/apimachinery/pkg/api/resource.Quantity</code></a>
</td>
<td>
   <p>weight 为该 ClusterQueue 或 Cohort 在 Cohort 中争夺未使用资源时提供比较优势。
份额基于每种资源超出名义配额的主导资源使用量，除以权重。
调度优先考虑来自份额最低的 ClusterQueue 和 Cohort 的工作负载，并抢占份额最高的。
权重为零意味着份额值为无穷大，这意味着该节点在与其他 ClusterQueue 和 Cohort 竞争时总是处于劣势。</p>
</td>
</tr>
</tbody>
</table>

## `FairSharingStatus`     {#kueue-x-k8s-io-v1beta1-FairSharingStatus}
    

**Appears in:**

- [ClusterQueueStatus](#kueue-x-k8s-io-v1beta1-ClusterQueueStatus)

- [CohortStatus](#kueue-x-k8s-io-v1beta1-CohortStatus)

- [LocalQueueStatus](#kueue-x-k8s-io-v1beta1-LocalQueueStatus)


<p>FairSharingStatus 包含有关当前公平共享状态的信息。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>weightedShare</code> <B>[Required]</B><br/>
<code>int64</code>
</td>
<td>
   <p>WeightedShare 表示节点所提供的所有资源中，超出名义配额的使用量与 Cohort 可借用资源的比值的最大值，再除以权重。
如果为零，表示节点的使用量低于名义配额。如果节点权重为零且正在借用，则返回 9223372036854775807，即最大可能的份额值。</p>
</td>
</tr>
<tr><td><code>admissionFairSharingStatus</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionFairSharingStatus"><code>AdmissionFairSharingStatus</code></a>
</td>
<td>
   <p>admissionFairSharingStatus 表示与 Admission Fair Sharing 相关的信息</p>
</td>
</tr>
</tbody>
</table>

## `FlavorFungibility`     {#kueue-x-k8s-io-v1beta1-FlavorFungibility}
    

**Appears in:**

- [ClusterQueueSpec](#kueue-x-k8s-io-v1beta1-ClusterQueueSpec)


<p>FlavorFungibility 决定工作负载在当前 flavor 借用或抢占前是否应尝试下一个 flavor。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>whenCanBorrow</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-FlavorFungibilityPolicy"><code>FlavorFungibilityPolicy</code></a>
</td>
<td>
   <p>whenCanBorrow 决定工作负载在当前 flavor 借用前是否应尝试下一个 flavor。可选值：</p>
<ul>
<li><code>Borrow</code>（默认）：如果可以借用，则在当前 flavor 分配。</li>
<li><code>TryNextFlavor</code>：即使当前 flavor 有足够资源可借用，也尝试下一个 flavor。</li>
</ul>
</td>
</tr>
<tr><td><code>whenCanPreempt</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-FlavorFungibilityPolicy"><code>FlavorFungibilityPolicy</code></a>
</td>
<td>
   <p>whenCanPreempt 决定工作负载在当前 flavor 抢占前是否应尝试下一个 flavor。可选值：</p>
<ul>
<li><code>Preempt</code>：如果可以抢占一些工作负载，则在当前 flavor 分配。</li>
<li><code>TryNextFlavor</code>（默认）：即使当前 flavor 有足够可抢占的候选，也尝试下一个 flavor。</li>
</ul>
</td>
</tr>
</tbody>
</table>

## `FlavorFungibilityPolicy`     {#kueue-x-k8s-io-v1beta1-FlavorFungibilityPolicy}
    
(Alias of `string`)

**Appears in:**

- [FlavorFungibility](#kueue-x-k8s-io-v1beta1-FlavorFungibility)


<p>FlavorFungibilityPolicy defines the policy for a flavor.
FlavorFungibilityPolicy 定义 flavor 的策略。</p>




## `FlavorQuotas`     {#kueue-x-k8s-io-v1beta1-FlavorQuotas}
    

**Appears in:**

- [ResourceGroup](#kueue-x-k8s-io-v1beta1-ResourceGroup)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ResourceFlavorReference"><code>ResourceFlavorReference</code></a>
</td>
<td>
   <p>name of this flavor. The name should match the .metadata.name of a
ResourceFlavor. If a matching ResourceFlavor does not exist, the
ClusterQueue will have an Active condition set to False.
此 flavor 的名称。名称应与 ResourceFlavor 的 .metadata.name 匹配。如果不存在匹配的 ResourceFlavor，则 ClusterQueue 的 Active 状态将为 False。</p>
</td>
</tr>
<tr><td><code>resources</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ResourceQuota"><code>[]ResourceQuota</code></a>
</td>
<td>
   <p>resources is the list of quotas for this flavor per resource.
There could be up to 16 resources.</p>
</td>
</tr>
</tbody>
</table>

## `FlavorUsage`     {#kueue-x-k8s-io-v1beta1-FlavorUsage}
    

**Appears in:**

- [ClusterQueueStatus](#kueue-x-k8s-io-v1beta1-ClusterQueueStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ResourceFlavorReference"><code>ResourceFlavorReference</code></a>
</td>
<td>
   <p>flavor 的名称。</p>
</td>
</tr>
<tr><td><code>resources</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ResourceUsage"><code>[]ResourceUsage</code></a>
</td>
<td>
   <p>resources 列出此 flavor 中资源的配额使用情况。</p>
</td>
</tr>
</tbody>
</table>

## `KubeConfig`     {#kueue-x-k8s-io-v1beta1-KubeConfig}
    

**Appears in:**

- [MultiKueueClusterSpec](#kueue-x-k8s-io-v1beta1-MultiKueueClusterSpec)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>location</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>KubeConfig 的位置。</p>
<p>如果 LocationType 为 Secret，则 Location 是 kueue 控制器管理器所在命名空间中的 Secret 名称。配置应存储在 &quot;kubeconfig&quot; 键中。</p>
</td>
</tr>
<tr><td><code>locationType</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-LocationType"><code>LocationType</code></a>
</td>
<td>
   <p>KubeConfig 位置的类型。</p>
</td>
</tr>
</tbody>
</table>

## `LocalQueueFlavorStatus`     {#kueue-x-k8s-io-v1beta1-LocalQueueFlavorStatus}
    

**Appears in:**

- [LocalQueueStatus](#kueue-x-k8s-io-v1beta1-LocalQueueStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ResourceFlavorReference"><code>ResourceFlavorReference</code></a>
</td>
<td>
   <p>name of the flavor.
flavor 的名称。</p>
</td>
</tr>
<tr><td><code>resources</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcename-v1-core"><code>[]k8s.io/api/core/v1.ResourceName</code></a>
</td>
<td>
   <p>resources used in the flavor.
flavor 中使用的资源。</p>
</td>
</tr>
<tr><td><code>nodeLabels</code><br/>
<code>map[string]string</code>
</td>
<td>
   <p>nodeLabels are labels that associate the ResourceFlavor with Nodes that
have the same labels.
nodeLabels 是将 ResourceFlavor 与具有相同标签的节点关联的标签。</p>
</td>
</tr>
<tr><td><code>nodeTaints</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#taint-v1-core"><code>[]k8s.io/api/core/v1.Taint</code></a>
</td>
<td>
   <p>nodeTaints are taints that the nodes associated with this ResourceFlavor
have.
nodeTaints 是与此 ResourceFlavor 关联的节点所具有的污点。</p>
</td>
</tr>
<tr><td><code>topology</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-TopologyInfo"><code>TopologyInfo</code></a>
</td>
<td>
   <p>topology is the topology that associated with this ResourceFlavor.</p>
<p>This is an alpha field and requires enabling the TopologyAwareScheduling
feature gate.</p>
<p>topology 是与此 ResourceFlavor 关联的拓扑。
这是一个 alpha 字段，需要启用 TopologyAwareScheduling 特性门控。</p>
</td>
</tr>
</tbody>
</table>

## `LocalQueueFlavorUsage`     {#kueue-x-k8s-io-v1beta1-LocalQueueFlavorUsage}
    

**Appears in:**

- [LocalQueueStatus](#kueue-x-k8s-io-v1beta1-LocalQueueStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ResourceFlavorReference"><code>ResourceFlavorReference</code></a>
</td>
<td>
   <p>name of the flavor.
flavor 的名称。</p>
</td>
</tr>
<tr><td><code>resources</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-LocalQueueResourceUsage"><code>[]LocalQueueResourceUsage</code></a>
</td>
<td>
   <p>resources lists the quota usage for the resources in this flavor.
resources 列出了此 flavor 中资源的配额使用情况。</p>
</td>
</tr>
</tbody>
</table>

## `LocalQueueName`     {#kueue-x-k8s-io-v1beta1-LocalQueueName}
    
(Alias of `string`)

**Appears in:**

- [WorkloadSpec](#kueue-x-k8s-io-v1beta1-WorkloadSpec)


<p>LocalQueueName is the name of the LocalQueue.
它必须是 DNS（RFC 1123）格式，最大长度为 253 个字符。</p>
<p>它必须是 DNS（RFC 1123）格式，最大长度为 253 个字符。</p>




## `LocalQueueResourceUsage`     {#kueue-x-k8s-io-v1beta1-LocalQueueResourceUsage}
    

**Appears in:**

- [LocalQueueFlavorUsage](#kueue-x-k8s-io-v1beta1-LocalQueueFlavorUsage)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcename-v1-core"><code>k8s.io/api/core/v1.ResourceName</code></a>
</td>
<td>
   <p>name of the resource.
资源的名称。</p>
</td>
</tr>
<tr><td><code>total</code> <B>[Required]</B><br/>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/api/resource#Quantity"><code>k8s.io/apimachinery/pkg/api/resource.Quantity</code></a>
</td>
<td>
   <p>total is the total quantity of used quota.
total 是已用配额的总量。</p>
</td>
</tr>
</tbody>
</table>

## `LocalQueueSpec`     {#kueue-x-k8s-io-v1beta1-LocalQueueSpec}
    

**Appears in:**

- [LocalQueue](#kueue-x-k8s-io-v1beta1-LocalQueue)


<p>LocalQueueSpec 定义了 LocalQueue 的期望状态</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>clusterQueue</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ClusterQueueReference"><code>ClusterQueueReference</code></a>
</td>
<td>
   <p>clusterQueue 是指向支持此 localQueue 的 clusterQueue 的引用。</p>
</td>
</tr>
<tr><td><code>stopPolicy</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-StopPolicy"><code>StopPolicy</code></a>
</td>
<td>
   <p>stopPolicy - 如果设置为非 None 的值，则 LocalQueue 被视为非活动状态，不会进行新的预留。
根据其值，相关的工作负载将：</p>
<ul>
<li>None - 工作负载被接纳</li>
<li>HoldAndDrain - 已接纳的工作负载会被驱逐，预留中的工作负载会取消预留</li>
<li>Hold - 已接纳的工作负载会运行至完成，预留中的工作负载会取消预留</li>
</ul>
</td>
</tr>
<tr><td><code>fairSharing</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-FairSharing"><code>FairSharing</code></a>
</td>
<td>
   <p>fairSharing 定义了 LocalQueue 在参与 AdmissionFairSharing 时的属性。仅当 Kueue 配置中启用 AdmissionFairSharing 时，这些值才有意义。</p>
</td>
</tr>
</tbody>
</table>

## `LocalQueueStatus`     {#kueue-x-k8s-io-v1beta1-LocalQueueStatus}
    

**Appears in:**

- [LocalQueue](#kueue-x-k8s-io-v1beta1-LocalQueue)


<p>LocalQueueStatus defines the observed state of LocalQueue
LocalQueueStatus 定义了 LocalQueue 的观测状态</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>pendingWorkloads</code><br/>
<code>int32</code>
</td>
<td>
   <p>PendingWorkloads is the number of Workloads in the LocalQueue not yet admitted to a ClusterQueue
PendingWorkloads 是 LocalQueue 中尚未被接纳到 ClusterQueue 的工作负载数量</p>
</td>
</tr>
<tr><td><code>reservingWorkloads</code><br/>
<code>int32</code>
</td>
<td>
   <p>reservingWorkloads is the number of workloads in this LocalQueue
reserving quota in a ClusterQueue and that haven't finished yet.
reservingWorkloads 是此 LocalQueue 中正在 ClusterQueue 预留配额且尚未完成的工作负载数量。</p>
</td>
</tr>
<tr><td><code>admittedWorkloads</code><br/>
<code>int32</code>
</td>
<td>
   <p>admittedWorkloads is the number of workloads in this LocalQueue
admitted to a ClusterQueue and that haven't finished yet.
admittedWorkloads 是此 LocalQueue 中已被接纳到 ClusterQueue 且尚未完成的工作负载数量。</p>
</td>
</tr>
<tr><td><code>conditions</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta"><code>[]k8s.io/apimachinery/pkg/apis/meta/v1.Condition</code></a>
</td>
<td>
   <p>Conditions hold the latest available observations of the LocalQueue
current state.
Conditions 保存了 LocalQueue 当前状态的最新可用观测信息。</p>
</td>
</tr>
<tr><td><code>flavorsReservation</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-LocalQueueFlavorUsage"><code>[]LocalQueueFlavorUsage</code></a>
</td>
<td>
   <p>flavorsReservation are the reserved quotas, by flavor currently in use by the
workloads assigned to this LocalQueue.
flavorsReservation 是分配给此 LocalQueue 的工作负载当前使用的 flavor 的预留配额。</p>
</td>
</tr>
<tr><td><code>flavorUsage</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-LocalQueueFlavorUsage"><code>[]LocalQueueFlavorUsage</code></a>
</td>
<td>
   <p>flavorsUsage are the used quotas, by flavor currently in use by the
workloads assigned to this LocalQueue.
flavorsUsage 是分配给此 LocalQueue 的工作负载当前使用的 flavor 的已用配额。</p>
</td>
</tr>
<tr><td><code>flavors</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-LocalQueueFlavorStatus"><code>[]LocalQueueFlavorStatus</code></a>
</td>
<td>
   <p>flavors lists all currently available ResourceFlavors in specified ClusterQueue.
flavors 列出了指定 ClusterQueue 中当前可用的所有 ResourceFlavor。</p>
</td>
</tr>
<tr><td><code>fairSharing</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-FairSharingStatus"><code>FairSharingStatus</code></a>
</td>
<td>
   <p>FairSharing contains the information about the current status of fair sharing.
FairSharing 包含有关当前公平共享状态的信息。</p>
</td>
</tr>
</tbody>
</table>

## `LocationType`     {#kueue-x-k8s-io-v1beta1-LocationType}
    
(Alias of `string`)

**Appears in:**

- [KubeConfig](#kueue-x-k8s-io-v1beta1-KubeConfig)





## `MultiKueueClusterSpec`     {#kueue-x-k8s-io-v1beta1-MultiKueueClusterSpec}
    

**Appears in:**

- [MultiKueueCluster](#kueue-x-k8s-io-v1beta1-MultiKueueCluster)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>kubeConfig</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-KubeConfig"><code>KubeConfig</code></a>
</td>
<td>
   <p>连接集群的信息。</p>
</td>
</tr>
</tbody>
</table>

## `MultiKueueClusterStatus`     {#kueue-x-k8s-io-v1beta1-MultiKueueClusterStatus}
    

**Appears in:**

- [MultiKueueCluster](#kueue-x-k8s-io-v1beta1-MultiKueueCluster)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>conditions</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta"><code>[]k8s.io/apimachinery/pkg/apis/meta/v1.Condition</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `MultiKueueConfigSpec`     {#kueue-x-k8s-io-v1beta1-MultiKueueConfigSpec}
    

**Appears in:**

- [MultiKueueConfig](#kueue-x-k8s-io-v1beta1-MultiKueueConfig)


<p>MultiKueueConfigSpec 定义了 MultiKueueConfig 的期望状态。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>clusters</code> <B>[Required]</B><br/>
<code>[]string</code>
</td>
<td>
   <p>ClusterQueue 中的工作负载应分发到的 MultiKueueCluster 名称列表。</p>
</td>
</tr>
</tbody>
</table>

## `Parameter`     {#kueue-x-k8s-io-v1beta1-Parameter}
    
(Alias of `string`)

**Appears in:**

- [ProvisioningRequestConfigSpec](#kueue-x-k8s-io-v1beta1-ProvisioningRequestConfigSpec)


<p>Parameter 限制为 255 个字符。</p>




## `PodSet`     {#kueue-x-k8s-io-v1beta1-PodSet}
    

**Appears in:**

- [WorkloadSpec](#kueue-x-k8s-io-v1beta1-WorkloadSpec)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-PodSetReference"><code>PodSetReference</code></a>
</td>
<td>
   <p>name 是 PodSet 的名称。</p>
</td>
</tr>
<tr><td><code>template</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podtemplatespec-v1-core"><code>k8s.io/api/core/v1.PodTemplateSpec</code></a>
</td>
<td>
   <p>template 是 Pod 模板。</p>
<p>template.metadata 中只允许标签和注解。</p>
<p>如果容器或 initContainer 的请求被省略，
它们默认为容器或 initContainer 的限制。</p>
<p>在准入时，用于过滤可以分配给此 podSet 的 ResourceFlavors 的规则
是 nodeSelector 和 nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
中与 ResourceFlavors 考虑的 nodeLabels 匹配的键。</p>
</td>
</tr>
<tr><td><code>count</code> <B>[Required]</B><br/>
<code>int32</code>
</td>
<td>
   <p>count 是规范中的 Pod 数量。</p>
</td>
</tr>
<tr><td><code>minCount</code><br/>
<code>int32</code>
</td>
<td>
   <p>minCount 是规范中接受的 Pod 最小数量
如果工作负载支持部分接纳。</p>
<p>如果未提供，则当前 PodSet 的部分接纳未启用。</p>
<p>工作负载中只能有一个 PodSet 使用此字段。</p>
<p>这是一个 alpha 字段，需要启用 PartialAdmission 功能门。</p>
</td>
</tr>
<tr><td><code>topologyRequest</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-PodSetTopologyRequest"><code>PodSetTopologyRequest</code></a>
</td>
<td>
   <p>topologyRequest 定义了 PodSet 的拓扑请求。</p>
</td>
</tr>
</tbody>
</table>

## `PodSetAssignment`     {#kueue-x-k8s-io-v1beta1-PodSetAssignment}
    

**Appears in:**

- [Admission](#kueue-x-k8s-io-v1beta1-Admission)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-PodSetReference"><code>PodSetReference</code></a>
</td>
<td>
   <p>Name 是 podSet 的名称。它应该与 .spec.podSets 中的名称之一匹配。</p>
</td>
</tr>
<tr><td><code>flavors</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-ResourceFlavorReference"><code>map[ResourceName]ResourceFlavorReference</code></a>
</td>
<td>
   <p>Flavors 为工作负载分配了每个资源的口味。</p>
</td>
</tr>
<tr><td><code>resourceUsage</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcelist-v1-core"><code>k8s.io/api/core/v1.ResourceList</code></a>
</td>
<td>
   <p>resourceUsage 跟踪工作负载中所有 Pod 所需的资源总量。</p>
<p>除了 podSet 规范中提供的内容外，此计算还考虑了准入时的 LimitRange 默认值和 RuntimeClass 开销。
配额回收时此字段不会改变。</p>
</td>
</tr>
<tr><td><code>count</code><br/>
<code>int32</code>
</td>
<td>
   <p>count 是准入时考虑的 Pod 数量。
配额回收时此字段不会改变。
如果此字段在添加此字段之前创建的 Workload，则可能缺少值，
在这种情况下，spec.podSets[*].count 值将用于。</p>
</td>
</tr>
<tr><td><code>topologyAssignment</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-TopologyAssignment"><code>TopologyAssignment</code></a>
</td>
<td>
   <p>topologyAssignment 表示拓扑分配，分为拓扑域，对应拓扑的最低级别。
分配指定每个拓扑域中要调度的 Pod 数量，并指定每个拓扑域的节点选择器，
如下所示：节点选择器键由 levels 字段指定（所有域都相同），
对应的节点选择器值由 domains.values 子字段指定。如果 TopologySpec.Levels 字段包含
&quot;kubernetes.io/hostname&quot; 标签，则 topologyAssignment 仅包含此标签的数据，
并省略拓扑中的更高级别</p>
<p>示例：</p>
<p>topologyAssignment:
levels:</p>
<ul>
<li>cloud.provider.com/topology-block</li>
<li>cloud.provider.com/topology-rack
domains:</li>
<li>values: [block-1, rack-1]
count: 4</li>
<li>values: [block-1, rack-2]
count: 2</li>
</ul>
<p>这里：</p>
<ul>
<li>4 个 Pod 要调度到匹配节点选择器的节点：
cloud.provider.com/topology-block: block-1
cloud.provider.com/topology-rack: rack-1</li>
<li>2 个 Pod 要调度到匹配节点选择器的节点：
cloud.provider.com/topology-block: block-1
cloud.provider.com/topology-rack: rack-2</li>
</ul>
<p>示例：
下面是一个等效于上述示例的示例，假设 Topology
对象将 kubernetes.io/hostname 定义为拓扑的最低级别。
因此，我们省略了拓扑中的更高级别，因为主机名标签
足以明确识别一个合适的节点。</p>
<p>topologyAssignment:
levels:</p>
<ul>
<li>kubernetes.io/hostname
domains:</li>
<li>values: [hostname-1]
count: 4</li>
<li>values: [hostname-2]
count: 2</li>
</ul>
</td>
</tr>
<tr><td><code>delayedTopologyRequest</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-DelayedTopologyRequestState"><code>DelayedTopologyRequestState</code></a>
</td>
<td>
   <p>delayedTopologyRequest 表示拓扑请求被延迟。
如果使用了 ProvisioningRequest AdmissionCheck，则拓扑分配可能被延迟。
Kueue 对每个 workload 调度第二个调度周期，其中至少有一个 PodSet
具有 delayedTopologyRequest=true 且没有 topologyAssignment。</p>
</td>
</tr>
</tbody>
</table>

## `PodSetReference`     {#kueue-x-k8s-io-v1beta1-PodSetReference}
    
(Alias of `string`)

**Appears in:**

- [PodSet](#kueue-x-k8s-io-v1beta1-PodSet)

- [PodSetAssignment](#kueue-x-k8s-io-v1beta1-PodSetAssignment)

- [PodSetRequest](#kueue-x-k8s-io-v1beta1-PodSetRequest)

- [PodSetUpdate](#kueue-x-k8s-io-v1beta1-PodSetUpdate)

- [ReclaimablePod](#kueue-x-k8s-io-v1beta1-ReclaimablePod)


<p>PodSetReference 是 PodSet 的名称。</p>




## `PodSetRequest`     {#kueue-x-k8s-io-v1beta1-PodSetRequest}
    

**Appears in:**

- [WorkloadStatus](#kueue-x-k8s-io-v1beta1-WorkloadStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-PodSetReference"><code>PodSetReference</code></a>
</td>
<td>
   <p>name 是 podSet 的名称。它应该与 .spec.podSets 中的名称之一匹配。</p>
</td>
</tr>
<tr><td><code>resources</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcelist-v1-core"><code>k8s.io/api/core/v1.ResourceList</code></a>
</td>
<td>
   <p>resources 是 podSet 中所有 Pod 所需的资源总量。</p>
<p>除了 podSet 规范中提供的内容外，此值还考虑了准入时的 LimitRange 默认值和 RuntimeClass 开销，
以及 resource.excludeResourcePrefixes 和 resource.transformations 的应用。</p>
</td>
</tr>
</tbody>
</table>

## `PodSetTopologyRequest`     {#kueue-x-k8s-io-v1beta1-PodSetTopologyRequest}
    

**Appears in:**

- [PodSet](#kueue-x-k8s-io-v1beta1-PodSet)


<p>PodSetTopologyRequest 定义了 PodSet 的拓扑请求。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>required</code><br/>
<code>string</code>
</td>
<td>
   <p>required 表示 PodSet 所需的拓扑级别，由 <code>kueue.x-k8s.io/podset-required-topology</code> PodSet
注解指示。</p>
</td>
</tr>
<tr><td><code>preferred</code><br/>
<code>string</code>
</td>
<td>
   <p>preferred 表示 PodSet 偏好的拓扑级别，由 <code>kueue.x-k8s.io/podset-preferred-topology</code> PodSet
注解指示。</p>
</td>
</tr>
<tr><td><code>unconstrained</code><br/>
<code>bool</code>
</td>
<td>
   <p>unconstrained 表示 Kueue 在完全可用容量内调度 PodSet 时没有约束，
没有对放置紧凑性的约束。这由 <code>kueue.x-k8s.io/podset-unconstrained-topology</code> PodSet
注解指示。</p>
</td>
</tr>
<tr><td><code>podIndexLabel</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>PodIndexLabel 表示 Pod 的索引标签名称。
例如，在 kubernetes job 中是：kubernetes.io/job-completion-index
JobSet 中是：kubernetes.io/job-completion-index（从 Job 继承）</p>
</td>
</tr>
<tr><td><code>subGroupIndexLabel</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>SubGroupIndexLabel 表示 PodSet 中复制的 Job（组）实例的索引标签名称。
例如，在 JobSet 中是 jobset.sigs.k8s.io/job-index。</p>
</td>
</tr>
<tr><td><code>subGroupCount</code> <B>[Required]</B><br/>
<code>int32</code>
</td>
<td>
   <p>SubGroupIndexLabel 表示 PodSet 中复制的 Job（组）数量。
例如，在 JobSet 中，此值从 jobset.sigs.k8s.io/replicatedjob-replicas 读取。</p>
</td>
</tr>
<tr><td><code>podSetSliceRequiredTopology</code><br/>
<code>string</code>
</td>
<td>
   <p>PodSetSliceRequiredTopology 表示 PodSet 切片的拓扑级别要求，由 <code>kueue.x-k8s.io/podset-slice-required-topology</code>
注解指示。</p>
</td>
</tr>
<tr><td><code>podSetSliceSize</code><br/>
<code>int32</code>
</td>
<td>
   <p>PodSetSliceSize 表示 PodSet 中一个子组（subgroup）的 Pod 数量，
其中 Kueue 在 <code>kueue.x-k8s.io/podset-slice-required-topology</code> 注解中定义的拓扑域级别上找到
请求的拓扑域。</p>
</td>
</tr>
</tbody>
</table>

## `PodSetUpdate`     {#kueue-x-k8s-io-v1beta1-PodSetUpdate}
    

**Appears in:**

- [AdmissionCheckState](#kueue-x-k8s-io-v1beta1-AdmissionCheckState)


<p>PodSetUpdate 包含 AdmissionChecks 建议的 PodSet 修改。
修改应仅限于添加 - 修改现有键或由多个 AdmissionChecks 提供相同键的修改是不允许的，
并且在工作负载接纳期间会导致失败。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-PodSetReference"><code>PodSetReference</code></a>
</td>
<td>
   <p>要修改的 PodSet 的名称。应与 Workload 的 PodSets 之一匹配。</p>
</td>
</tr>
<tr><td><code>labels</code><br/>
<code>map[string]string</code>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
<tr><td><code>annotations</code><br/>
<code>map[string]string</code>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
<tr><td><code>nodeSelector</code><br/>
<code>map[string]string</code>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
<tr><td><code>tolerations</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core"><code>[]k8s.io/api/core/v1.Toleration</code></a>
</td>
<td>
   <span class="text-muted">No description provided.</span></td>
</tr>
</tbody>
</table>

## `PreemptionPolicy`     {#kueue-x-k8s-io-v1beta1-PreemptionPolicy}
    
(Alias of `string`)

**Appears in:**

- [ClusterQueuePreemption](#kueue-x-k8s-io-v1beta1-ClusterQueuePreemption)


<p>PreemptionPolicy defines the preemption policy.
PreemptionPolicy 定义抢占策略。</p>




## `ProvisioningRequestConfigPodSetMergePolicy`     {#kueue-x-k8s-io-v1beta1-ProvisioningRequestConfigPodSetMergePolicy}
    
(Alias of `string`)

**Appears in:**

- [ProvisioningRequestConfigSpec](#kueue-x-k8s-io-v1beta1-ProvisioningRequestConfigSpec)





## `ProvisioningRequestConfigSpec`     {#kueue-x-k8s-io-v1beta1-ProvisioningRequestConfigSpec}
    

**Appears in:**

- [ProvisioningRequestConfig](#kueue-x-k8s-io-v1beta1-ProvisioningRequestConfig)


<p>ProvisioningRequestConfigSpec 定义了 ProvisioningRequestConfig 的期望状态</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>provisioningClassName</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>ProvisioningClassName 描述了资源调度的不同模式。
详情请参考 autoscaling.x-k8s.io ProvisioningRequestSpec.ProvisioningClassName。</p>
</td>
</tr>
<tr><td><code>parameters</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-Parameter"><code>map[string]Parameter</code></a>
</td>
<td>
   <p>Parameters 包含类可能需要的所有其他参数。</p>
</td>
</tr>
<tr><td><code>managedResources</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcename-v1-core"><code>[]k8s.io/api/core/v1.ResourceName</code></a>
</td>
<td>
   <p>managedResources 包含由自动扩缩容管理的资源列表。</p>
<p>如果为空，则认为所有资源都被管理。</p>
<p>如果不为空，ProvisioningRequest 只会包含请求了其中至少一个资源的 podset。</p>
<p>如果所有 workload 的 podset 都没有请求任何被管理的资源，则认为该 workload 已就绪。</p>
</td>
</tr>
<tr><td><code>retryStrategy</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-ProvisioningRequestRetryStrategy"><code>ProvisioningRequestRetryStrategy</code></a>
</td>
<td>
   <p>retryStrategy 定义了重试 ProvisioningRequest 的策略。
如果为 null，则应用默认配置，参数如下：
backoffLimitCount:  3
backoffBaseSeconds: 60 - 1 分钟
backoffMaxSeconds:  1800 - 30 分钟</p>
<p>若要关闭重试机制，将 retryStrategy.backoffLimitCount 设为 0。</p>
</td>
</tr>
<tr><td><code>podSetUpdates</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-ProvisioningRequestPodSetUpdates"><code>ProvisioningRequestPodSetUpdates</code></a>
</td>
<td>
   <p>podSetUpdates 指定 workload 的 PodSetUpdates 更新，用于定位已调度节点。</p>
</td>
</tr>
<tr><td><code>podSetMergePolicy</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-ProvisioningRequestConfigPodSetMergePolicy"><code>ProvisioningRequestConfigPodSetMergePolicy</code></a>
</td>
<td>
   <p>podSetMergePolicy 指定在传递给集群自动扩缩容器前合并 PodSet 的策略。</p>
</td>
</tr>
</tbody>
</table>

## `ProvisioningRequestPodSetUpdates`     {#kueue-x-k8s-io-v1beta1-ProvisioningRequestPodSetUpdates}
    

**Appears in:**

- [ProvisioningRequestConfigSpec](#kueue-x-k8s-io-v1beta1-ProvisioningRequestConfigSpec)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>nodeSelector</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-ProvisioningRequestPodSetUpdatesNodeSelector"><code>[]ProvisioningRequestPodSetUpdatesNodeSelector</code></a>
</td>
<td>
   <p>nodeSelector 指定 NodeSelector 的更新列表。</p>
</td>
</tr>
</tbody>
</table>

## `ProvisioningRequestPodSetUpdatesNodeSelector`     {#kueue-x-k8s-io-v1beta1-ProvisioningRequestPodSetUpdatesNodeSelector}
    

**Appears in:**

- [ProvisioningRequestPodSetUpdates](#kueue-x-k8s-io-v1beta1-ProvisioningRequestPodSetUpdates)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>key</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>key 指定 NodeSelector 的键。</p>
</td>
</tr>
<tr><td><code>valueFromProvisioningClassDetail</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>valueFromProvisioningClassDetail 指定 ProvisioningRequest.status.provisioningClassDetails 的键，
其值用于更新。</p>
</td>
</tr>
</tbody>
</table>

## `ProvisioningRequestRetryStrategy`     {#kueue-x-k8s-io-v1beta1-ProvisioningRequestRetryStrategy}
    

**Appears in:**

- [ProvisioningRequestConfigSpec](#kueue-x-k8s-io-v1beta1-ProvisioningRequestConfigSpec)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>backoffLimitCount</code><br/>
<code>int32</code>
</td>
<td>
   <p>BackoffLimitCount 定义了最大重试次数。
达到该次数后，workload 会被停用（<code>.spec.activate</code>=<code>false</code>）。</p>
<p>每次退避时间大约为 &quot;b*2^(n-1)+Rand&quot;，其中：</p>
<ul>
<li>&quot;b&quot; 由 &quot;BackoffBaseSeconds&quot; 参数设定，</li>
<li>&quot;n&quot; 为 &quot;workloadStatus.requeueState.count&quot;，</li>
<li>&quot;Rand&quot; 为随机抖动。
在此期间，workload 被视为不可接受，其他 workload 有机会被调度。
默认连续重排队延迟约为：(60s, 120s, 240s, ...)。</li>
</ul>
<p>默认值为 3。</p>
</td>
</tr>
<tr><td><code>backoffBaseSeconds</code><br/>
<code>int32</code>
</td>
<td>
   <p>BackoffBaseSeconds 定义了被驱逐 workload 重新排队的指数退避基数。</p>
<p>默认值为 60。</p>
</td>
</tr>
<tr><td><code>backoffMaxSeconds</code><br/>
<code>int32</code>
</td>
<td>
   <p>BackoffMaxSeconds 定义了被驱逐 workload 重新排队的最大退避时间。</p>
<p>默认值为 1800。</p>
</td>
</tr>
</tbody>
</table>

## `QueueingStrategy`     {#kueue-x-k8s-io-v1beta1-QueueingStrategy}
    
(Alias of `string`)

**Appears in:**

- [ClusterQueueSpec](#kueue-x-k8s-io-v1beta1-ClusterQueueSpec)





## `ReclaimablePod`     {#kueue-x-k8s-io-v1beta1-ReclaimablePod}
    

**Appears in:**

- [WorkloadStatus](#kueue-x-k8s-io-v1beta1-WorkloadStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-PodSetReference"><code>PodSetReference</code></a>
</td>
<td>
   <p>name 是 PodSet 的名称。</p>
</td>
</tr>
<tr><td><code>count</code> <B>[Required]</B><br/>
<code>int32</code>
</td>
<td>
   <p>count 是请求资源不再需要的 Pod 数量。</p>
</td>
</tr>
</tbody>
</table>

## `RequeueState`     {#kueue-x-k8s-io-v1beta1-RequeueState}
    

**Appears in:**

- [WorkloadStatus](#kueue-x-k8s-io-v1beta1-WorkloadStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>count</code><br/>
<code>int32</code>
</td>
<td>
   <p>count 记录工作负载被重队列的次数
当 deactivated (<code>.spec.activate</code>=<code>false</code>) 工作负载被 reactivated (<code>.spec.activate</code>=<code>true</code>) 时，
此计数将重置为 null。</p>
</td>
</tr>
<tr><td><code>requeueAt</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta"><code>k8s.io/apimachinery/pkg/apis/meta/v1.Time</code></a>
</td>
<td>
   <p>requeueAt 记录工作负载将被重队列的时间。
当 deactivated (<code>.spec.activate</code>=<code>false</code>) 工作负载被 reactivated (<code>.spec.activate</code>=<code>true</code>) 时，
此时间将重置为 null。</p>
</td>
</tr>
</tbody>
</table>

## `ResourceFlavorReference`     {#kueue-x-k8s-io-v1beta1-ResourceFlavorReference}
    
(Alias of `string`)

**Appears in:**

- [AdmissionCheckStrategyRule](#kueue-x-k8s-io-v1beta1-AdmissionCheckStrategyRule)

- [FlavorQuotas](#kueue-x-k8s-io-v1beta1-FlavorQuotas)

- [FlavorUsage](#kueue-x-k8s-io-v1beta1-FlavorUsage)

- [LocalQueueFlavorStatus](#kueue-x-k8s-io-v1beta1-LocalQueueFlavorStatus)

- [LocalQueueFlavorUsage](#kueue-x-k8s-io-v1beta1-LocalQueueFlavorUsage)

- [PodSetAssignment](#kueue-x-k8s-io-v1beta1-PodSetAssignment)


<p>ResourceFlavorReference 是 ResourceFlavor 的名称。</p>




## `ResourceFlavorSpec`     {#kueue-x-k8s-io-v1beta1-ResourceFlavorSpec}
    

**Appears in:**

- [ResourceFlavor](#kueue-x-k8s-io-v1beta1-ResourceFlavor)


<p>ResourceFlavorSpec defines the desired state of the ResourceFlavor
ResourceFlavorSpec 定义了 ResourceFlavor 的期望状态</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>nodeLabels</code><br/>
<code>map[string]string</code>
</td>
<td>
   <p>nodeLabels are labels that associate the ResourceFlavor with Nodes that
have the same labels.
当 Workload 被接纳时，其 podsets 只能分配到 nodeLabels 匹配 nodeSelector 和 nodeAffinity 字段的 ResourceFlavors。
一旦 ResourceFlavor 被分配给 podSet，集成 Workload 对象的控制器应将 ResourceFlavor 的 nodeLabels 注入到 Workload 的 pods 中。</p>
<p>nodeLabels 最多可以有 8 个元素。</p>
</td>
</tr>
<tr><td><code>nodeTaints</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#taint-v1-core"><code>[]k8s.io/api/core/v1.Taint</code></a>
</td>
<td>
   <p>nodeTaints are taints that the nodes associated with this ResourceFlavor
have.
Workloads 的 podsets 必须对这些 nodeTaints 有容忍（tolerations），才能在接纳时分配到该 ResourceFlavor。
当此 ResourceFlavor 也设置了匹配的容忍（在 .spec.tolerations 中），则在接纳时不会考虑 nodeTaints。
只评估 'NoSchedule' 和 'NoExecute' 污点效果，忽略 'PreferNoSchedule'。</p>
<p>nodeTaints 示例：
cloud.provider.com/preemptible=&quot;true&quot;:NoSchedule</p>
<p>nodeTaints 最多可以有 8 个元素。</p>
</td>
</tr>
<tr><td><code>tolerations</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core"><code>[]k8s.io/api/core/v1.Toleration</code></a>
</td>
<td>
   <p>tolerations are extra tolerations that will be added to the pods admitted in
the quota associated with this resource flavor.</p>
<p>容忍（toleration）示例：
cloud.provider.com/preemptible=&quot;true&quot;:NoSchedule</p>
<p>tolerations 最多可以有 8 个元素。</p>
</td>
</tr>
<tr><td><code>topologyName</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-TopologyReference"><code>TopologyReference</code></a>
</td>
<td>
   <p>topologyName indicates topology for the TAS ResourceFlavor.
topologyName 表示 TAS ResourceFlavor 的拓扑。
指定后，会从与 Resource Flavor nodeLabels 匹配的节点中抓取拓扑信息。</p>
</td>
</tr>
</tbody>
</table>

## `ResourceGroup`     {#kueue-x-k8s-io-v1beta1-ResourceGroup}
    

**Appears in:**

- [ClusterQueueSpec](#kueue-x-k8s-io-v1beta1-ClusterQueueSpec)

- [CohortSpec](#kueue-x-k8s-io-v1beta1-CohortSpec)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>coveredResources</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcename-v1-core"><code>[]k8s.io/api/core/v1.ResourceName</code></a>
</td>
<td>
   <p>coveredResources 是此组中 flavors 覆盖的资源列表。
例如：cpu、memory、vendor.com/gpu。
此列表不能为空，且最多包含 16 个资源。</p>
</td>
</tr>
<tr><td><code>flavors</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-FlavorQuotas"><code>[]FlavorQuotas</code></a>
</td>
<td>
   <p>flavors 是为此组资源提供配额的 flavor 列表。
通常，不同的 flavor 代表不同的硬件型号（如 gpu 型号、cpu 架构）或定价模式（按需 vs 竞价 cpu）。
每个 flavor 必须以与 .resources 字段相同的顺序列出此组的所有资源。
此列表不能为空，且最多包含 16 个 flavor。</p>
</td>
</tr>
</tbody>
</table>

## `ResourceQuota`     {#kueue-x-k8s-io-v1beta1-ResourceQuota}
    

**Appears in:**

- [FlavorQuotas](#kueue-x-k8s-io-v1beta1-FlavorQuotas)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcename-v1-core"><code>k8s.io/api/core/v1.ResourceName</code></a>
</td>
<td>
   <p>name of this resource.
此资源的名称。</p>
</td>
</tr>
<tr><td><code>nominalQuota</code> <B>[Required]</B><br/>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/api/resource#Quantity"><code>k8s.io/apimachinery/pkg/api/resource.Quantity</code></a>
</td>
<td>
   <p>nominalQuota is the quantity of this resource that is available for
Workloads admitted by this ClusterQueue at a point in time.
nominalQuota 必须为非负数。
nominalQuota 应代表集群中可用于运行作业的资源（扣除系统组件和非 kueue 管理的 pod 所消耗的资源后）。在自动扩缩的集群中，nominalQuota 应考虑如 Kubernetes cluster-autoscaler 等组件可提供的资源。</p>
<p>If the ClusterQueue belongs to a cohort, the sum of the quotas for each
(flavor, resource) combination defines the maximum quantity that can be
allocated by a ClusterQueue in the cohort.
如果 ClusterQueue 属于 cohort，则每个（flavor, resource）组合的配额总和定义了 cohort 中 ClusterQueue 可分配的最大数量。</p>
</td>
</tr>
<tr><td><code>borrowingLimit</code><br/>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/api/resource#Quantity"><code>k8s.io/apimachinery/pkg/api/resource.Quantity</code></a>
</td>
<td>
   <p>borrowingLimit is the maximum amount of quota for the [flavor, resource]
combination that this ClusterQueue is allowed to borrow from the unused
quota of other ClusterQueues in the same cohort.
borrowingLimit 是该 ClusterQueue 允许从同一 cohort 中其他 ClusterQueue 未使用配额中借用的 [flavor, resource] 组合的最大配额。
In total, at a given time, Workloads in a ClusterQueue can consume a
quantity of quota equal to nominalQuota+borrowingLimit, assuming the other
ClusterQueues in the cohort have enough unused quota.
总体上，在某一时刻，ClusterQueue 中的工作负载可以消耗等于 nominalQuota+borrowingLimit 的配额，前提是 cohort 中其他 ClusterQueue 有足够的未使用配额。
If null, it means that there is no borrowing limit.
如果为 null，表示没有借用上限。
If not null, it must be non-negative.
如果不为 null，则必须为非负数。
borrowingLimit must be null if spec.cohort is empty.
如果 spec.cohort 为空，则 borrowingLimit 必须为 null。</p>
</td>
</tr>
<tr><td><code>lendingLimit</code><br/>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/api/resource#Quantity"><code>k8s.io/apimachinery/pkg/api/resource.Quantity</code></a>
</td>
<td>
   <p>lendingLimit is the maximum amount of unused quota for the [flavor, resource]
combination that this ClusterQueue can lend to other ClusterQueues in the same cohort.
lendingLimit 是该 ClusterQueue 可以借给同一 cohort 中其他 ClusterQueue 的 [flavor, resource] 组合的最大未使用配额。
In total, at a given time, ClusterQueue reserves for its exclusive use
a quantity of quota equals to nominalQuota - lendingLimit.
总体上，在某一时刻，ClusterQueue 为其专用保留的配额等于 nominalQuota - lendingLimit。
If null, it means that there is no lending limit, meaning that
all the nominalQuota can be borrowed by other clusterQueues in the cohort.
如果为 null，表示没有出借上限，意味着所有 nominalQuota 都可以被 cohort 中其他 ClusterQueue 借用。
If not null, it must be non-negative.
如果不为 null，则必须为非负数。
lendingLimit must be null if spec.cohort is empty.
如果 spec.cohort 为空，则 lendingLimit 必须为 null。
This field is in beta stage and is enabled by default.
此字段为 beta 阶段，默认启用。</p>
</td>
</tr>
</tbody>
</table>

## `ResourceUsage`     {#kueue-x-k8s-io-v1beta1-ResourceUsage}
    

**Appears in:**

- [FlavorUsage](#kueue-x-k8s-io-v1beta1-FlavorUsage)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcename-v1-core"><code>k8s.io/api/core/v1.ResourceName</code></a>
</td>
<td>
   <p>资源的名称</p>
</td>
</tr>
<tr><td><code>total</code> <B>[Required]</B><br/>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/api/resource#Quantity"><code>k8s.io/apimachinery/pkg/api/resource.Quantity</code></a>
</td>
<td>
   <p>total 是已用配额的总量，包括从 cohort 借用的数量。</p>
</td>
</tr>
<tr><td><code>borrowed</code> <B>[Required]</B><br/>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/api/resource#Quantity"><code>k8s.io/apimachinery/pkg/api/resource.Quantity</code></a>
</td>
<td>
   <p>Borrowed 是从 cohort 借用的配额数量。换句话说，是超出 nominalQuota 的已用配额。</p>
</td>
</tr>
</tbody>
</table>

## `SchedulingStats`     {#kueue-x-k8s-io-v1beta1-SchedulingStats}
    

**Appears in:**

- [WorkloadStatus](#kueue-x-k8s-io-v1beta1-WorkloadStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>evictions</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-WorkloadSchedulingStatsEviction"><code>[]WorkloadSchedulingStatsEviction</code></a>
</td>
<td>
   <p>evictions 跟踪按原因和底层原因的驱逐统计。</p>
</td>
</tr>
</tbody>
</table>

## `StopPolicy`     {#kueue-x-k8s-io-v1beta1-StopPolicy}
    
(Alias of `string`)

**Appears in:**

- [ClusterQueueSpec](#kueue-x-k8s-io-v1beta1-ClusterQueueSpec)

- [LocalQueueSpec](#kueue-x-k8s-io-v1beta1-LocalQueueSpec)





## `TopologyAssignment`     {#kueue-x-k8s-io-v1beta1-TopologyAssignment}
    

**Appears in:**

- [PodSetAssignment](#kueue-x-k8s-io-v1beta1-PodSetAssignment)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>levels</code> <B>[Required]</B><br/>
<code>[]string</code>
</td>
<td>
   <p>levels 是拓扑分配中拓扑级别的有序列表（即节点标签键），从最高到最低。</p>
</td>
</tr>
<tr><td><code>domains</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-TopologyDomainAssignment"><code>[]TopologyDomainAssignment</code></a>
</td>
<td>
   <p>domains 是拓扑分配的列表，按拓扑域在拓扑的最低级别划分。</p>
</td>
</tr>
</tbody>
</table>

## `TopologyDomainAssignment`     {#kueue-x-k8s-io-v1beta1-TopologyDomainAssignment}
    

**Appears in:**

- [TopologyAssignment](#kueue-x-k8s-io-v1beta1-TopologyAssignment)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>values</code> <B>[Required]</B><br/>
<code>[]string</code>
</td>
<td>
   <p>values 是描述拓扑域的节点选择器值的有序列表。
值对应连续的拓扑级别，从最高到最低。</p>
</td>
</tr>
<tr><td><code>count</code> <B>[Required]</B><br/>
<code>int32</code>
</td>
<td>
   <p>count 表示在 values 字段指示的拓扑域中要调度的 Pod 数量。</p>
</td>
</tr>
</tbody>
</table>

## `TopologyInfo`     {#kueue-x-k8s-io-v1beta1-TopologyInfo}
    

**Appears in:**

- [LocalQueueFlavorStatus](#kueue-x-k8s-io-v1beta1-LocalQueueFlavorStatus)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>name</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-TopologyReference"><code>TopologyReference</code></a>
</td>
<td>
   <p>name is the name of the topology.</p>
<p>name 是拓扑的名称。</p>
</td>
</tr>
<tr><td><code>levels</code> <B>[Required]</B><br/>
<code>[]string</code>
</td>
<td>
   <p>levels define the levels of topology.</p>
<p>levels 定义了拓扑的层级。</p>
</td>
</tr>
</tbody>
</table>

## `TopologyReference`     {#kueue-x-k8s-io-v1beta1-TopologyReference}
    
(Alias of `string`)

**Appears in:**

- [ResourceFlavorSpec](#kueue-x-k8s-io-v1beta1-ResourceFlavorSpec)

- [TopologyInfo](#kueue-x-k8s-io-v1beta1-TopologyInfo)


<p>TopologyReference is the name of the Topology.
TopologyReference 是拓扑的名称。</p>




## `WorkloadSchedulingStatsEviction`     {#kueue-x-k8s-io-v1beta1-WorkloadSchedulingStatsEviction}
    

**Appears in:**

- [SchedulingStats](#kueue-x-k8s-io-v1beta1-SchedulingStats)



<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>reason</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>reason 指定驱逐原因的程序化标识符。</p>
</td>
</tr>
<tr><td><code>underlyingCause</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>underlyingCause 提供更详细的解释，补充驱逐原因。
这可能是空字符串。</p>
</td>
</tr>
<tr><td><code>count</code> <B>[Required]</B><br/>
<code>int32</code>
</td>
<td>
   <p>count 跟踪此原因和详细原因的驱逐次数。</p>
</td>
</tr>
</tbody>
</table>

## `WorkloadSpec`     {#kueue-x-k8s-io-v1beta1-WorkloadSpec}
    

**Appears in:**

- [Workload](#kueue-x-k8s-io-v1beta1-Workload)


<p>WorkloadSpec 定义了 Workload 的期望状态</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>podSets</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-PodSet"><code>[]PodSet</code></a>
</td>
<td>
   <p>podSets 是一组同质 Pod 的集合，每个由 Pod 规范和数量描述。
必须至少有一个元素，最多 8 个。
podSets 不可更改。</p>
</td>
</tr>
<tr><td><code>queueName</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-LocalQueueName"><code>LocalQueueName</code></a>
</td>
<td>
   <p>queueName 是 Workload 关联的 LocalQueue 的名称。
当 .status.admission 不为 null 时，queueName 不可更改。</p>
</td>
</tr>
<tr><td><code>priorityClassName</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>如果指定，表示 workload 的优先级。
&quot;system-node-critical&quot; 和 &quot;system-cluster-critical&quot; 是两个特殊关键字，分别表示最高和次高优先级。
其他名称必须通过创建具有该名称的 PriorityClass 对象来定义。
如果未指定，workload 的优先级将为默认值或零（如果没有默认值）。</p>
</td>
</tr>
<tr><td><code>priority</code> <B>[Required]</B><br/>
<code>int32</code>
</td>
<td>
   <p>Priority 决定了在 ClusterQueue 中访问资源的顺序。
优先级值由 PriorityClassName 填充。
值越高，优先级越高。
如果指定了 priorityClassName，则 priority 不能为空。</p>
</td>
</tr>
<tr><td><code>priorityClassSource</code> <B>[Required]</B><br/>
<code>string</code>
</td>
<td>
   <p>priorityClassSource 决定 priorityClass 字段是指 pod PriorityClass 还是 kueue.x-k8s.io/workloadpriorityclass。
Workload 的 PriorityClass 可以接受 pod priorityClass 或 workloadPriorityClass 的名称。
当使用 pod PriorityClass 时，priorityClassSource 字段值为 scheduling.k8s.io/priorityclass。</p>
</td>
</tr>
<tr><td><code>active</code> <B>[Required]</B><br/>
<code>bool</code>
</td>
<td>
   <p>Active 决定 workload 是否可以被接纳到队列中。
将 active 从 true 改为 false 会驱逐所有正在运行的 workload。
可能的值有：</p>
<ul>
<li>false: 表示 workload 永远不会被接纳，并驱逐正在运行的 workload</li>
<li>true: 表示 workload 可以被评估是否接纳到其所属队列。</li>
</ul>
<p>默认为 true</p>
</td>
</tr>
<tr><td><code>maximumExecutionTimeSeconds</code><br/>
<code>int32</code>
</td>
<td>
   <p>maximumExecutionTimeSeconds 如果提供，表示 workload 最多可被接纳的时间（秒），超时后会自动被停用。</p>
<p>如果未指定，则不对 Workload 强制执行执行时间限制。</p>
</td>
</tr>
</tbody>
</table>

## `WorkloadStatus`     {#kueue-x-k8s-io-v1beta1-WorkloadStatus}
    

**Appears in:**

- [Workload](#kueue-x-k8s-io-v1beta1-Workload)


<p>WorkloadStatus 定义了 Workload 的观察状态。</p>


<table class="table">
<thead><tr><th width="30%">Field</th><th>Description</th></tr></thead>
<tbody>
    
  
<tr><td><code>admission</code> <B>[Required]</B><br/>
<a href="#kueue-x-k8s-io-v1beta1-Admission"><code>Admission</code></a>
</td>
<td>
   <p>admission 持有工作负载被集群队列接纳的参数。admission 可以设置为 null，
但一旦设置，其字段不能更改。</p>
</td>
</tr>
<tr><td><code>requeueState</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-RequeueState"><code>RequeueState</code></a>
</td>
<td>
   <p>requeueState 持有工作负载在 PodsReadyTimeout 原因下被驱逐时的重队列状态。</p>
</td>
</tr>
<tr><td><code>conditions</code><br/>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta"><code>[]k8s.io/apimachinery/pkg/apis/meta/v1.Condition</code></a>
</td>
<td>
   <p>conditions 持有工作负载的最新可用观察结果。</p>
<p>条件的类型可以是：</p>
<ul>
<li>Admitted: 工作负载通过集群队列被接纳。</li>
<li>Finished: 关联的工作负载运行完成（失败或成功）。</li>
<li>PodsReady: 至少 <code>.spec.podSets[*].count</code> Pods 已就绪或成功。</li>
</ul>
</td>
</tr>
<tr><td><code>reclaimablePods</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-ReclaimablePod"><code>[]ReclaimablePod</code></a>
</td>
<td>
   <p>reclaimablePods 跟踪资源预留不再需要的 Pod 数量。</p>
</td>
</tr>
<tr><td><code>admissionChecks</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-AdmissionCheckState"><code>[]AdmissionCheckState</code></a>
</td>
<td>
   <p>admissionChecks 列出工作负载所需的所有接纳检查及其当前状态。</p>
</td>
</tr>
<tr><td><code>resourceRequests</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-PodSetRequest"><code>[]PodSetRequest</code></a>
</td>
<td>
   <p>resourceRequests 提供了资源请求的详细视图，
这些资源在非接纳工作负载被考虑接纳时请求。
如果 admission 不为 null，则 resourceRequests 将为空，
因为 admission.resourceUsage 包含详细信息。</p>
</td>
</tr>
<tr><td><code>accumulatedPastExexcutionTimeSeconds</code><br/>
<code>int32</code>
</td>
<td>
   <p>accumulatedPastExexcutionTimeSeconds 持有工作负载在 Admitted 状态中花费的总时间（秒），
在之前的 <code>Admit</code> - <code>Evict</code> 周期中。</p>
</td>
</tr>
<tr><td><code>schedulingStats</code><br/>
<a href="#kueue-x-k8s-io-v1beta1-SchedulingStats"><code>SchedulingStats</code></a>
</td>
<td>
   <p>schedulingStats 跟踪调度统计信息</p>
</td>
</tr>
</tbody>
</table>
  