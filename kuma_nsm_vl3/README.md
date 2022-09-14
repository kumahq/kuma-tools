# Kuma - NSM (Network Service Mesh) vl3 integration 

This Example is used to show the working demo for Kuma - NSM vl3 integration.

For more info about NSM go to: [networkservicemesh.io](https://networkservicemesh.io/)

For more info about this demo go to: [Demo presentation](https://docs.google.com/presentation/d/12aiunkKqPLOe1R0o_QUdkuZM4LPGhN0g9AwJ1vuFpOo/edit?usp=sharing)

## Requires

Set KUBECONFIG1 and KUBECONFIG2 to your cluster config paths accordingly.
- [Load balancer](https://github.com/networkservicemesh/deployments-k8s/blob/fb11bedb58d46b406e277aac289f48eb0137035a/examples/interdomain/loadbalancer/README.md)
- [Interdomain DNS](https://github.com/networkservicemesh/deployments-k8s/blob/fb11bedb58d46b406e277aac289f48eb0137035a/examples/interdomain/dns/README.md)
- [Interdomain spire](https://github.com/networkservicemesh/deployments-k8s/blob/c5351a86850adc9014693853b7fafe83e65c20b8/examples/interdomain/spire/README.md)
- [Interdomain nsm](https://github.com/networkservicemesh/deployments-k8s/blob/c5351a86850adc9014693853b7fafe83e65c20b8/examples/interdomain/nsm/README.md)

## Run
1. Start vl3
```bash
kubectl --kubeconfig=$KUBECONFIG1 apply -k ./vl3-dns
kubectl --kubeconfig=$KUBECONFIG1 -n ns-dns-vl3 wait --for=condition=ready --timeout=2m pod -l app=vl3-ipam
```

2. Install kumactl

Install kumactl by following [Kuma docs](https://kuma.io/docs/1.7.x/installation/kubernetes/)
```bash
curl -L https://kuma.io/installer.sh | VERSION=1.7.0 ARCH=amd64 bash -
export PATH=$PWD/kuma-1.7.0/bin:$PATH
```

3. Create control-plane configuration
```bash
kumactl generate tls-certificate --hostname=control-plane-kuma.my-vl3-network --hostname=kuma-control-plane.kuma-system.svc --type=server --key-file=./tls.key --cert-file=./tls.crt
```
```bash
cp ./tls.crt ./ca.crt
```
```bash
kubectl --kubeconfig=$KUBECONFIG1 apply -f namespace.yaml
kubectl --kubeconfig=$KUBECONFIG1 create secret generic general-tls-certs --namespace=kuma-system --from-file=./tls.key --from-file=./tls.crt --from-file=./ca.crt
```
```bash
kumactl install control-plane --tls-general-secret=general-tls-certs --tls-general-ca-bundle=$(cat ./ca.crt | base64) > ./control-plane/control-plane.yaml
```

4. Start the control-plane on the first cluster
```bash
kubectl --kubeconfig=$KUBECONFIG1 apply -k ./control-plane
```

5. Start redis database with the sidecar on the first cluster
```bash
kubectl --kubeconfig=$KUBECONFIG1 apply -f demo-redis.yaml
kubectl --kubeconfig=$KUBECONFIG1 -n kuma-demo wait --for=condition=ready --timeout=3m pod -l app=redis
```

6. Start counter page with the sidecar on the second cluster
```bash
kubectl --kubeconfig=$KUBECONFIG2 apply -f demo-app.yaml
kubectl --kubeconfig=$KUBECONFIG2 -n kuma-demo wait --for=condition=ready --timeout=3m pod -l app=demo-app
```


7. Forward ports to open counter page
```bash
kubectl --kubeconfig=$KUBECONFIG2 port-forward svc/demo-app -n kuma-demo 5000:5000 &
```
```bash
response=$(curl -X POST localhost:5000/increment)
```

8. Check the response for no errors
```bash
echo $response | grep '"err":null'
```

You can also go to [locahost:5000](https://localhost:5000) to get the counter page and test the application yourself.

## Cleanup
```bash
kubectl --kubeconfig=$KUBECONFIG1 delete ns kuma-system kuma-demo ns-dns-vl3
kubectl --kubeconfig=$KUBECONFIG2 delete ns kuma-demo
rm tls.crt tls.key ca.crt
rm -rf kuma-1.7.0
```
