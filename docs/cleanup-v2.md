# Cleanup after KrakenD v3 upgrade

After all clusters run the v3 operator, the old v2 ConfigMaps are left behind. They are owned by the
`Krakend` resource, so they are not garbage collected, and the operator no longer manages them.

Per Krakend instance (`<fullname>` = `<name>-krakend`, or `<name>` if it already contains `krakend`):

- `<fullname>-config`
- `<fullname>-partials`
- `<fullname>-settings`
- `<fullname>-templates`

PodDisruptionBudgets and NetworkPolicies were disabled in the old chart, so there is nothing else to
remove. Old ReplicaSets are cleaned up by the Deployment's `revisionHistoryLimit`.

Run all steps per cluster (`kubectl config use-context ...`).

## 1. Verify all Krakends run v3

All Deployments should use the v3 image and the `-v3` ConfigMap, with `READY` = `WANT`:

```bash
kubectl get deploy -A -l app.kubernetes.io/name=krakend \
  -o custom-columns='NS:.metadata.namespace,NAME:.metadata.name,READY:.status.readyReplicas,WANT:.spec.replicas,IMAGE:.spec.template.spec.containers[0].image,CM:.spec.template.spec.volumes[*].configMap.name'
```

Pods and their status:

```bash
kubectl get pods -A -l app.kubernetes.io/name=krakend \
  -o custom-columns='NS:.metadata.namespace,POD:.metadata.name,PHASE:.status.phase,READY:.status.containerStatuses[0].ready,REASON:.status.containerStatuses[0].state.waiting.reason,IMAGE:.spec.containers[0].image'
```

No leftover Argo Rollouts from `deploymentType: rollout` (empty, or "doesn't have a resource type"):

```bash
kubectl get rollouts.argoproj.io -A -l app.kubernetes.io/name=krakend
```

## 2. List old ConfigMaps

ConfigMaps owned by a `Krakend`, except the new `-v3` ones:

```bash
kubectl get cm -A -o json | jq -r '.items[]
  | select(any(.metadata.ownerReferences[]?; .kind == "Krakend"))
  | select(.metadata.name | endswith("-v3") | not)
  | "\(.metadata.namespace) \(.metadata.name)"' | sort -u > old-cms.txt
cat old-cms.txt
```

## 3. Verify they are not in use

Collect every ConfigMap referenced by pods and workloads (volumes, projected volumes, `envFrom`, `env`).
ReplicaSets are left out on purpose: old Krakend ReplicaSets with 0 replicas still reference the old ConfigMaps.

```bash
kubectl get pods,deploy,sts,ds,jobs,cronjobs -A -o json | jq -r '.items[] | .metadata.namespace as $ns
  | (.spec.jobTemplate.spec.template.spec // .spec.template.spec // .spec)
  | ( (.volumes[]? | .configMap.name, .projected.sources[]?.configMap.name),
      ((.containers[]?, .initContainers[]?) | .envFrom[]?.configMapRef.name, .env[]?.valueFrom.configMapKeyRef.name) )
  | select(. != null) | "\($ns) \(.)"' | sort -u > used-cms.txt

comm -12 old-cms.txt used-cms.txt   # must print nothing
```

If anything is printed, stop and find out what uses it.

## 4. Delete

**Destructive.** After this, `kubectl rollout undo` to a v2 ReplicaSet no longer works (pods would fail
to mount the missing ConfigMaps). Only run when rollback to v2 is no longer needed.

Dry run:

```bash
while read ns name; do kubectl delete cm -n "$ns" "$name" --dry-run=server; done < old-cms.txt
```

Delete:

```bash
while read ns name; do kubectl delete cm -n "$ns" "$name"; done < old-cms.txt
```
