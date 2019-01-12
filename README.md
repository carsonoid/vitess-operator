# vitess-operator

A Kubernetes operator for Vitess clusters.

## TODO

- [ ] Create vtctld Deployment and Service
- [ ] Create PodDisruptionBudgets
- [ ] Create ConfigMap
- [ ] Create vttablet service
- [ ] Create vtgate Deployment and Service
- [ ] Reconcile all the things

## Dev

- Install the [operator sdk](https://github.com/operator-framework/operator-sdk)
- Configure local kubectl access to a test cluster
- Create the CRDs in your test cluster
    - `kubectl create -f deploy/crds`
- Run the operator locally
    - `operator-sdk up local`
- Create a test cluster
    - `kubectl create -f _examples/vitess_v1alpha2_vitesscluster_simple.yaml`
