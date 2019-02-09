# vitess-operator

A Kubernetes operator for Vitess clusters.

## TODO

- [x] Create a StatefulSet for each VitessTablet in a VitessCluster
- [x] Create a Job to elect the initial master in each VitessShard
- [X] Fix parenting and normalization
- [x] Create vtctld Deployment and Service
- [ ] Create PodDisruptionBudgets
- [ ] Create ConfigMap
- [ ] Create vttablet service
- [ ] Create vtgate Deployment and Service
- [ ] Reconcile all the things!
- [ ] Move shard master election into the operator

## Dev

- Install the [operator sdk](https://github.com/operator-framework/operator-sdk)
- Configure local kubectl access to a test Kubernetes cluster
- Create the CRDs in your Kubernetes cluster
    - `kubectl create -f deploy/crds`
- Run the operator locally
    - `operator-sdk up local`
- Create the etcd servers
    - `kubectl create -f _examples/etcd-clusters.yaml`
- Create a sample cluster with everything in one resource
    - `kubectl create -f _examples/all-in-one.yaml`
