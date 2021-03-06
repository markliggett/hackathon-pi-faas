# HACKATHON 2019 - FaaS on Bosun

Hackathon attempt at building a primative autoscaling serverless app served by [kNative](https://knative.dev) on [Bosun](https://github.com/bazaarvoice/bosun)

What I did in advance:
 - Installed Istio and Knative onto a running Bosun cluster following https://istio.io/docs/setup/kubernetes/install/kubernetes/ and https://knative.dev/v0.3-docs/install/

What this project contains:
  - A primative go-lang web app called `myfirstserverlessapp` which includes 3 delegating methods that have tracing enabled (Openjaeger/Zipkin), it also supports the passing of an `COLOUR` environment variable that is used in the app response.
  - A multi-stage docker build that creates a primative go-lang web-app container
  - Some Kubernetes files that publish the web-app to a knative-enabled cluster
  - Tooling around calling the loading the web-app in order for it to autoscale (from zero to 10 replicas)
  - Tooling and options around traffic engineering, in this instance the percentage of traffic that is routed to the app based on its version

## PRE-REQS

For building and publishing the container _(optional)_:
- `docker` - to create an app image _(optional)_

For deploying | testing | loading:
- `kubectl` - to talk to the Bosun cluster (with admin privileges if a new namespaces is required)
- `jq` - JSON convenience (see https://github.com/stedolan/jq)
- `hey`- Simple web-app loading tool (see https://github.com/rakyll/hey), you'll need a valid go-lang installation to use this tool.

## BUILDING

### Build the app image _(optional)_
This build uses a multi-stage build, where a container is used to build the app before it is copied to a `scratch` container
```
docker build . -t <my-dockerhub-username>/myfirstserverlessapp:v0.1
```
If successful you'll end up with a small docker image containing the web-app
```
REPOSITORY                                          TAG                 IMAGE ID            CREATED             SIZE
<my-dockerhub-username>/myfirstserverlessapp       v0.1               0904135b244d        8 minutes ago       12.6MB
```

### Publish the image to dockerhub _(optional)_
```
docker push <my-dockerhub-username>/myfirstserverlessapp:v0.1
```

## DEPLOYING

### Create an Istio enabled Kubernetes namespace (Admins only!)
```
kubectl apply -f hackathon-pi-knative-ns.yml
```
Alternatively label an existing namespace for Istio Injection.
```
kubectl label namespace <my-namespace> istio-injection=enabled
```

_NOTE: For this hackathon the namespace `hackathon-pi-knative` is used._

### Deploy the app using knative
```
kubectl apply -f hackathon-pi-knative-svc.yml
```

## TESTING

### Export the hostname running the Istio gateway
```
export KNATIVE_HOST=$(kubectl get svc istio-ingressgateway --namespace istio-system -o json | jq -r .status.loadBalancer.ingress[0].hostname)
```

### Export the knative route domain
This value is passed in the HTTP header and used by Istio to redirect the request to the right app.
```
export KNATIVE_DOMAIN=$(kubectl get route myfirstserverlessapp -o json | jq -r .status.domain)
```

### Make a request to the serverless app (via the Istio gateway)

The app expects 3 parameters:

| Name        | Use           | Expects  |
| ------------- |:-------------:| -----:|
| `in` | get the highest prime found below this number | `int` |
| `nap` | a nap period in seconds | `int` =< 10 |
| `processes` | number of concurrent `go` processes to used to calculate Pi (badly) | `int` |

```
curl -H "Host: ${KNATIVE_DOMAIN}" "http://${KNATIVE_HOST}/?in=10000&nap=3&processes=50"
```

With a bit of luck the FaaS should respond, eg;

![alt text](images/blue-app.png "Blue coloured app")

## AUTOSCALING

The knative service is configured with a artifical target concurrency of 1 - knative should scale the app (by adding more replicas) if it recieves more than 1 concurrent request.  Approximately 30 seconds after requests stop, knative will scale the app to zero instances.

View the current app replicas:
```
kubectl config set-context $(kubectl config current-context) --namespace=hackathon-pi-knative
kubectl get pods
```
If there's no requests (and it's been deployed for more than 3 minutes), you should see zero replicas:
```
$ kubectl get pods
No resources found.
```

Load the app for 2 minutes using (knative takes a minute to establish avg concurrency):
```
cd go/src
export GOPATH=$(pwd)
go get -u github.com/rakyll/hey
hey -z 120s -c 11 -host ${KNATIVE_DOMAIN} "http://${KNATIVE_HOST}/?in=100000&nap=3&processes=5000"
```

Knative should autoscale the replicas (up to a max of 10):
```
kubectl get pods
NAME                                                     READY   STATUS    RESTARTS   AGE
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-5vnxt   0/3     Pending   0          1m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-97mn4   3/3     Running   0          1m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-hth4c   3/3     Running   0          1m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-jnc74   0/3     Pending   0          1m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-k6gtn   3/3     Running   0          1m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-l6bmm   0/3     Pending   0          1m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-pgtd6   0/3     Pending   0          1m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-r4586   0/3     Pending   0          1m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-rbvkk   0/3     Pending   0          1m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-rx2sc   0/3     Pending   0          1m
```

You can also see the scaling in the Grafana graphs:
```
kubectl port-forward --namespace knative-monitoring $(kubectl get pods --namespace knative-monitoring \
--selector=app=grafana --output=jsonpath="{.items..metadata.name}") 3000
```
Then navigate to http://localhost:3000 and look at the `Deployment` dashboard:
![alt text](images/grafana-scaling.png "Grafana showings replica count increase under load")

Approximately 30 seconds after loading the web-app you will see the replicas scale to zero:
```
kubectl get pods
NAME                                                     READY   STATUS            RESTARTS   AGE
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-97mn4   3/3     Running           0          3m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-hth4c   2/3     Terminating       0          3m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-k6gtn   2/3     Terminating       0          3m
myfirstserverlessapp-9mq4l-deployment-7f5cd97f44-rx2sc   2/3     Terminating       0          3m
```

## TRAFFIC ENGINEERING

Knative can take advantage of the Istio service mesh to engineer traffic routed to the serverless app.

First remove the running Knative service used above (if applicable):
```
kubectl delete -f hackathon-pi-knative-svc.yml
```

Create a `v1` configuration for the app:
```
kubectl apply -f hackathon-pi-config-v1.yml
```

Once accepted this will generate a configuration with a unique `revision` for the application.  The `revision` can be found by calling:
```
kubectl describe config myfirstserverlessapp
```

Traffic can be routed to this particular `revision` using a Knative route.  Modify the file `k8s/hackathon-pi-route.yml` to suit the generated `revision` then apply using:
```
kubectl apply -f hackathon-pi-route.yml
```

At this point, sending a Postman/browser request to the serverless app will result in a blue response (same as the image above).

Once confirmed create a new `v2` configuration.  Note, no traffic will be sent to the `v2` configuration as no route is configured.
```
kubectl apply -f hackathon-pi-config-v1.yml
```

Determine the new `revision` in the same manner as earlier:
```
kubectl describe config myfirstserverlessapp
```

Traffic can now be directed to the revision by editing the existing route (either via a direct `kubectl edit route myfirstserverlessapp` or by modifying `k8s/hackathon-pi-route.yml` and re-applying).  For example:
```yaml
<output-snipped>
  traffic:
  - revisionName: v1Placeholder
    percent: 50
    name: v1
  - revisionName: v1Placeholder
    percent: 50
    name: v2
```

After applying this change, sending a Postman/browser request to the serverless app will result in a logo colour change with every other request:

![alt text](images/green-app.png "Green coloured app")

More useful Knative revision information can be found here - https://github.com/knative/serving/blob/master/docs/spec/normative_examples.md