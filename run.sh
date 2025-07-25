set -ex
kind delete cluster -n koord
kind create cluster --config ./kind.yaml -n koord --kubeconfig ~/.kube/kind-koord --image registry.cn-hangzhou.aliyuncs.com/acejilam/node:v1.30.3
linux-install-moniter.sh
#kubectl taint node koord-worker2 spot-taint:NoSchedule
#kubectl taint node koord-worker3 spot-taint:NoSchedule

docker pull registry.cn-hangzhou.aliyuncs.com/ls-2018/mygo:v1.24.1
kind load docker-image -n koord registry.cn-hangzhou.aliyuncs.com/ls-2018/mygo:v1.24.1
make kind-image-build
make importer-image
make kueueviz-image
make load-image
kubectl create ns kueue-system
kubectl apply -f ./deploy --server-side

until kubectl -n kueue-system get secret kueue-webhook-server-cert -oyaml |grep "tls.key" ; do
  sleep 1
done

kubectl rollout restart deployment kueue-controller-manager -n kueue-system
kubectl wait --for=condition=Ready -A --all pod --timeout=30s || true 

sleep 5
# 需要等待 secret 的内容 挂载到pod
kubectl apply -f "examples/1、preposition.yaml"
kubectl apply -f "examples/2、job.yaml"

