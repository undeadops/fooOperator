# FooOperator

[![Go Report Card](https://goreportcard.com/badge/github.com/undeadops/fooOperator)](https://goreportcard.com/report/github.com/undeadops/fooOperator)
![Go Coverage](./coverage_badge.png)
[![GoDoc](https://godoc.org/github.com/undeadops/fooOperator?status.svg)](https://pkg.go.dev/mod/github.com/undeadops/fooOperator)
[![License](https://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)

This was an experimental Operator, and rewritten with a pattern outlined via an example project: https://github.com/openshift/gcp-project-operator

# What It Does

Its designed to supply a dedicated Pod, Service (svc), and Ingress.  Normally not something that is commonly done, but there are a couple use cases, like a headless browser, or needing to launch a bunch of game servers.

It does this through two CRDs.  `Foo` and the `FooBar`.  `Foo`, is the base level object, that owns the Pod, Service and Ingress.  The `FooBar`, acts like a Deployment object, its role is to create a bunch of `Foos`.

In the grander scheme, an API server would be required to allocate these `Foos` out to the requesting parties.

Lets take a look at what the configurations look like

## Configuring

```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: fooworker
  namespace: barx
automountServiceAccountToken: true
---
apiVersion: apps.undeadops.xyz/v1alpha1
kind: Foo
metadata:
  name: simplyfoo
  namespace: barx
  annotations:
    prometheus.io/scrap: "true"
    prometheus.io/port: "9797"
  labels:
    owner: "sre"
spec:
  pod:
    serviceAccountName: fooworker
    image: ghcr.io/stefanprodan/podinfo:6.1.6
    ports:
    - name: http
      containerPort: 9898
      protocol: TCP
    - name: http-metrics
      containerPort: 9797
      protocol: TCP
    - name: grpc
      containerPort: 9999
      protocol: TCP
    command:
    - ./podinfo
    - --port=9898
    - --port-metrics=9797
    - --grpc-port=9999
    - --grpc-service-name=podinfo
    - --level=info
    - --random-delay=false
    - --random-error=false
    env:
    - name: PODINFO_UI_COLOR
      value: "#34577c"
    resources:
      limits:
        cpu: 2000m
        memory: 512Mi
      requests:
        cpu: 100m
        memory: 64Mi
  ingress:
    servicePort: 9898
    ingressClassName: traefik
```

That is the equivalent of creating a "Pod" instead of a "Deployment".  Overall, its designed to be very similar to a Normal Pod definition.

A `FooBar` Definition looks like the following:

```yaml
apiVersion: apps.undeadops.xyz/v1alpha1
kind: FooBar
metadata:
  name: undead
  namespace: barx
  annotations:
    prometheus.io/scrap: "true"
    prometheus.io/port: "9797"
  labels:
    owner: "sre"
spec:
  replicas: 2
  foo:
    pod:
      image: ghcr.io/stefanprodan/podinfo:6.1.6
      ports:
        - name: http
          containerPort: 9898
          protocol: TCP
        - name: http-metrics
          containerPort: 9797
          protocol: TCP
        - name: grpc
          containerPort: 9999
          protocol: TCP
      command:
        - ./podinfo
        - --port=9898
        - --port-metrics=9797
        - --grpc-port=9999
        - --grpc-service-name=podinfo
        - --level=info
        - --random-delay=false
        - --random-error=false
      env:
        - name: PODINFO_UI_COLOR
          value: "#34577c"
      resources:
        limits:
          cpu: 2000m
          memory: 512Mi
        requests:
          cpu: 100m
          memory: 64Mi
    ingress:
      servicePort: 9898
      ingressClassName: traefik
```

From that, it should create 2 copies of the `Foo` resource.


