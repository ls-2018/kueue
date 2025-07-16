t(){
  docker pull $1
  kind load docker-image -n koord $1
}
t registry.k8s.io/kueue/kueue:v0.12.4
t registry.k8s.io/kueue/kueueviz-frontend:v0.12.4
t registry.k8s.io/kueue/kueueviz-backend:v0.12.4