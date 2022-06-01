# image-backup-controller
Image Backup Controller ensures all running deployments/daemonSets images belong to our backup registry, cloning all external images. 
Once cloned it updates resource spec and rollouts the new backup images.

## Description
We have the needing to watch Deployments and DaemonSets resources and spot what images are external to our backup registry.
On detected external image we clone it to the destination backup registry, and then we update resource Spec rolling out updated versions.

First development choice was to implement 2 separate controllers, watching deployments and daemonSets, both share the whole logic
presenting a unique friction point in their own types, they can be implemented in a generic way, sharing mainly everything as both share the whole logic.
This choice was done in the first implementation, but thinking more about it, this basic scenario has some corner cases apart from the lack of visibility as resources are moving and no state reflection in the system is shown.
Apart from that, on the situation that we need to increase concurrency, it could potentially happen that we will create an image backup more than once at the same time,
example:
Slow backup process is started, and we receive another event from the same resource, basic controller implementation it's not aware of current backup executions.

This opens the space to an 'operator' implementation where deployment/daemonSet controllers acts as producers from image backup tasks to be executed, image backup controller will take care on backup generation and reflect its progress through the status subresource
At the end this model works in a collaborative way, similar to Deployments/ReplicaSets/Pods relation as example, in that scenario our deployment/dameonset controllers will be able to watch image backup task progress through its state, being able to complete the rollout process once all resource backup tasks are completed.

The flow works as:
- deployment/daemonSet watches for objects on ready state from a non-restricted namespaces
- on deployment/daemonSet create/update event controller spots non-backup used image (initContainers/containers)
    - it checks if exists an image backup task related
        - if it's found continue checks execution state and continue
        - if none is found it will create an image backup task fot it
- ImageBackup controller process image backup executions progressing the CRD Status subresource
    - on execution success an expiration timer will take care of image backup removals


## Assumptions
Initial assumptions are:
- pods are already running in the cluster, so that Deployments/DaemonSets are expected to be Ready
- we are just interested in crete/update events
- restrict events from banned namespaces (kube-proxy)
- we are not interested in any other workload types (StatefulSets, Jobs, CronJobs...)

From the original assumptions we can define what are the events that we want to watch, predicates will implement each one of the restrictions
Overall idea is that we just execute Reconcile when an image backup must be generated and rollout

## Development Process
The whole project has been developed using Kubebuilder (3.4.1) and local Minikube (v1.25.2). 
Autogeneration has been widely use creating project scaffolding and manifests.
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster

#### Project scaffolding:
```
kubebuilder init --domain k8slab.io --repo github.com/marcosQuesada/image-backup-controller
kubebuilder create api --group k8slab.io --version v1alpha1 --kind Deployment
kubebuilder create api --group k8slab.io --version v1alpha1 --kind DaemonSet
kubebuilder create api --group k8slab.io --version v1alpha1 --kind ImageBackup
```

#### CRD generation
ImageBackup CRD definition, generate deepCopy and manifests
```
make generate
make manifests
```

#### Install CRD
Install CRDs on K8s cluster (I'm using local minikube)
```
make install
```

#### Run Locally
Controller can be run locally (ensure required backup registry credentials from env var to make it run locally)
```
make run
```

#### Docker image creation
```
make docker-build
make docker-push IMG=marcosquesada/image-backup-controller:latest
```

### Running on the cluster

#### Create Image Backup namespace
```
kubectl create ns image-backup
```

#### Create backup registry secrets (replace with your own credentials)
```
kubectl create secret generic backup-registry-secret --from-literal=username=xxxx --from-literal=passowrd=xxxx -n image-backup

```

#### Deploy controller

```
make deploy
```

#### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

#### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

### Test Samples
Example Deployment and DaemonSet are provided in config/samples/ folder, install them as:

```sh
kubectl apply -f config/samples/
```

#### Check test sample used images helper
Before backup image rollout:
```
kubectl get deployments nginx -n nginx -o jsonpath='{.spec.template.spec.containers[*].image}'
nginx:1.14.0

kubectl get daemonset fluentd -n fluentd -o jsonpath='{.spec.template.spec.containers[*].image}'
fluentd:latest
```

After image update rollout:
```
kubectl get deployments nginx -n nginx -o jsonpath='{.spec.template.spec.containers[*].image}'
docker.io/marcosquesada/library_nginx:1.14.0

kubectl get daemonset fluentd -n fluentd -o jsonpath='{.spec.template.spec.containers[*].image}'
docker.io/marcosquesada/library_fluentd:latest
```

## Further Improvements
- use autogenerated informer
- fire events (Recorder record.EventRecorder)
- improve BDD controller testing
- include meta conditions on ImageBackup CRD reflecting resource transitions

## TODO
- Grant Permissions to Prometheus Server so that it can scrape protected metrics
  - deploy prometheus stack and check the integration(pending)
```
kubectl create clusterrolebinding metrics --clusterrole=metrics-reader --serviceaccount=image-nackup:controller-manager
```

## Asciinema links
- Deployment example: https://asciinema.org/a/w3UQAtuttNZZp2Yrlue4hQBCa
    - crd view: https://asciinema.org/a/lLGWGw16PCOdi0Gee5GMf0jNJ
- Daemonset example: https://asciinema.org/a/8YPmVeqD7YvnBfjdpZW7NG986
  - crd view: https://asciinema.org/a/0ZUmNU61RMacibueavcwkTlPj
