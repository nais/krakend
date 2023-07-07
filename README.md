# krakend

## Testing in dev-nais-dev
Install krakend in dev-nais-dev:
```
helm install krakend .
```
Apply test app and oauth2 server:

```
kubectl apply -f hack/
```
Test the echo endpoint:

```
TOKEN=$(curl -s -d "grant_type=client_credentials&client_id=yolo&client_secret=bolo&scope=yolo" https://mock-oauth2-server.dev.dev-nais.cloud.nais.io/debugger/token  | jq -r '.access_token')
curl -H "Authorization: Bearer $TOKEN"  https://krakend.dev.dev-nais.cloud.nais.io/echo -v
```

## Notes

* Install krakend as a feature in nais-system
* Add default config, no endpoints, consider static documentation portal something
* krakend holds a list of endpoints in a configmap called krakend-partials
* Teams add configmap in each namespace with their own config, we create a watcher for the cms (by annotation/label or something)
* The watcher creates the neccessary endpoints in krakend based on the configmaps

