# image-backup-controller
// TODO(user): Add simple overview of use/purpose

## Description
// TODO(user): An in-depth paragraph about your project and overview of use

## Assumptions
 Initial assumptions are:
- pods are already running in the cluster, so that Deployments/DaemonSets are expected to be Ready
- we are just interested in crete/update events
- restrict events from banned namespaces (kube-proxy)
- we are not interested in any other workload types (StatefulSets, Jobs, CronJobs...)

From the original assumptions we can define what are the events that we want to watch, predicates will implement each one of the restrictions
Overall idea is that we just execute Reconcile when an image backup must be generated and rollout

## Development Notes
 The whole project has been developed using Kubebuilder (3.4.1), and so autogenerating scaffolding and manifests.
 
 First development choice was to implement 2 separate controllers, watching deployments and daemonSets.
 In both cases they are checking used images (initContainers and containers specs), and this was enough to execute backup generation against our backupRegistry.
 Once backup is completed it just updates deployment/daemonSets spec forcing rolling update.

 But, this basic scenario has some corner cases, on the situation that we need to increase concurrency, it could potentially happen that we will create an image backup more than once at the same time,
example: 
 Slow backup process is started, and we receive another event from the same resource, basic controller implementation it's not aware of current backup executions.

 This opens the space to a 'traditional operator' implementation where deployment/daemonSet controllers acts as producers from image backup tasks to be executed, image backup controller will take care on backup generation and reflect its progress through the status subresource
 At the end this model works in a collaborative way, similar to Deployments/ReplicaSets/Pods relation as example

The flow works as:
- deployment/daemonSet watches for objects on ready state from a non-restricted namespaces
- on deployment/daemonSet create/update event controller spots non-backup used image (initContainers/containers) 
  - it checks if exists an image backup task related
    - if it's found continue checks execution state and continue 
    - if none is found it will create an image backup task fot it
- ImageBackup controller process image backup executions progressing the CRD Status subresource
  - on execution success an expiration timer will take care of image backup removals

## Development Process

Project scaffolding:
```
kubebuilder init --domain k8slab.io --repo github.com/marcosQuesada/image-backup-controller
kubebuilder create api --group k8slab.io --version v1alpha1 --kind Deployment
kubebuilder create api --group k8slab.io --version v1alpha1 --kind DaemonSet
kubebuilder create api --group k8slab.io --version v1alpha1 --kind ImageBackup
```
Define ImageBackup CRD, generate deepCopy and manifests
```
make generate
make manifests
```

Install CRDs on K8s cluster (I'm using local minikube)
```
make install
```

## Improvements
- use autogenerated informer

## TODO
- env vars [deploy]
- add samples properly! (deployments, daemonset)
- metrics
- controller testing
- add some events
- Conditions?
- Clean credentials from tests !!!!
- SyncPeriod
- Logs to Debug

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=<some-registry>/image-backup-controller:tag
```
	
3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/image-backup-controller:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

