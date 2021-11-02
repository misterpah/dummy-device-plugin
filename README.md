# Sample Kuberentes Device Plugin

1. run shell command
2. use k9s to access container's shell
3. echo $foos

```shell
kubectl apply -f devicePlugin.yaml
kubectl apply -f example-pod1.yaml
kubectl apply -f example-pod2.yaml
kubectl apply -f example-pod3.yaml
```

## pushing changes to dockerhub
docker build . -t monsterpah/sample-device-plugin
docker push monsterpah/sample-device-plugin

## remark

in this repo, the most interesting item are `ListAndWatch` in `main.go`.
In real world, this function will probe/detect device/hw status and populate it into a k8s object (devicepluginv1beta1.Device).


## reference
https://github.com/fengye87/sample-device-plugin