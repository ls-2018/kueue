#!/usr/bin/env bash
kubectl --context kind-kind-manager --kubeconfig ~/.kube/kind-koord delete -f config/components/crd/bases --ignore-not-found
kubectl --context kind-kind-worker1 --kubeconfig ~/.kube/kind-koord delete -f config/components/crd/bases --ignore-not-found
kubectl --context kind-kind-worker2 --kubeconfig ~/.kube/kind-koord delete -f config/components/crd/bases --ignore-not-found




kubectl --context kind-kind-manager --kubeconfig ~/.kube/kind-koord apply -f config/components/crd/bases --server-side
kubectl --context kind-kind-worker1 --kubeconfig ~/.kube/kind-koord apply -f config/components/crd/bases --server-side
kubectl --context kind-kind-worker2 --kubeconfig ~/.kube/kind-koord apply -f config/components/crd/bases --server-side