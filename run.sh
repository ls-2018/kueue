install-k8s-by-kind.sh koord v1.30.3
docker pull registry.cn-hangzhou.aliyuncs.com/ls-2018/mygo:v1.24.1
kind load docker-image -n koord registry.cn-hangzhou.aliyuncs.com/ls-2018/mygo:v1.24.1
make kind-image-build
make importer-image
make kueueviz-image
make artifacts
make load-image
kubectl create ns kueue-system
kubectl apply -f ./artifacts --server-side
#kubectl delete -f ./deploy/
#kubectl apply --server-side -f ./deploy/
#
#
#kubectl apply -f examples/admin/single-clusterqueue-setup.yaml
##kubectl apply -f examples/pods-kueue/kueue-pod.yaml
#kubectl apply -f examples/job.yaml

