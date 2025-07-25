---

title: "使用 Kubeflow 运行"
linkTitle: "Kubeflow 作业"
weight: 6
date: 2023-08-23
description: 如何使用 Kubeflow 运行 Kueue
no_list: true
---

以下任务展示了如何运行由 Kueue 管理的 Kubeflow 作业。

### [MPI Operator](https://github.com/kubeflow/mpi-operator)集成 {#mpi-operator-integration}
- [运行由 Kueue 管理的 Kubeflow MPIJob](/zh-CN/docs/tasks/run_kubeflow_jobs/run_mpijobs)。

### [Trainer](https://github.com/kubeflow/trainer)集成 {#trainer-integration}

{{% alert title="注意" color="primary" %}}
Kueue 仅支持 Trainer v1.9.x 及之前的传统作业，不支持新的 TrainJob。
{{% /alert %}}

- [运行由 Kueue 管理的 Kubeflow PyTorchJob](/zh-CN/docs/tasks/run_kubeflow_jobs/run_pytorchjobs)。
- [运行由 Kueue 管理的 Kubeflow TFJob](/zh-CN/docs/tasks/run_kubeflow_jobs/run_tfjobs)。
- [运行由 Kueue 管理的 Kubeflow XGBoostJob](/zh-CN/docs/tasks/run_kubeflow_jobs/run_xgboostjobs)。
- [运行由 Kueue 管理的 Kubeflow PaddleJob](/zh-CN/docs/tasks/run_kubeflow_jobs/run_paddlejobs)。
- [运行由 Kueue 管理的 Kubeflow JAXJob](/zh-CN/docs/tasks/run_kubeflow_jobs/run_jaxjobs)。
