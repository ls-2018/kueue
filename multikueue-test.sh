t() {
	docker image inspect $1 >/dev/null 2>&1 || docker pull $1
}
t docker.io/kubeflow/training-operator:v1-3f15cb8
t docker.io/mpioperator/mpi-operator:0.6.0
t ghcr.io/kubeflow/training-v1/training-operator:v1-3f15cb8
t quay.io/ibm/appwrapper:v1.1.2
t registry.cn-hangzhou.aliyuncs.com/ls-2018/mygo:v1.24.1
t quay.io/kuberay/operator:v1.3.1

docker-empty-container.sh
kind delete cluster --name worker1
kind delete cluster --name worker2
git tag -d v0.12.4
git reset --soft ab6a91294
git add .
git commit -s -m "cn"
git tag v0.12.4
make kind-image-build
make importer-image
make kueueviz-image
make test-multikueue-e2e
