kubectl delete -f ./deploy/
kubectl apply --server-side -f ./deploy/

kubectl apply -f examples/admin/single-clusterqueue-setup.yaml
kubectl create -f examples/jobs/sample-job.yaml

ResourceFlavor 定了匹配的节点、容忍度等信息





CQ -> LQ -> Workload



cohort  -> CQ
            -> Flavor
            -> 抢占策略
        -> Flavor -> 可以出借