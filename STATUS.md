# Status
TODOs:
* Only update krakend (unstructured) resources if the spec has changed - see nais/replicator for inspiration
    * https://github.com/nais/replicator/commit/f9197f779919924648d1c5586817db69ba399376
    * https://github.com/nais/replicator/pull/11
* add readyness and liveness probes for krakend instances
* log with fields in operator
* add metrics and alerts
  * for the actual krakend deployment too
* add azure ad auth provider
* find some strategy for upgrading krakend image - i.e. dependabot
* add doc and examples for salesforce use case
* add netpol based on backendHost so we can remove appname? i.e. check if it is servicediscovery or not using both within same ns and with full svc url
* cleanup hack/samples directory with use cases, use debugapp as sample app.
* more tests 
* add doc for debugging krakend and troubleshooting/faq
* logging: indexed fields in kibana - ref https://nav-it.slack.com/archives/C02C9GJP17D/p1698228702439199?thread_ts=1697806246.535379&cid=C02C9GJP17D
* metrics for krakend instance: https://www.krakend.io/docs/telemetry/prometheus/
* tweaking resource requirements: https://www.krakend.io/docs/deploying/server-dimensioning/


Feature requests:
* investigate ingress with subdomain: whatever.sub.ekstern.dev.nav.no gives cloud.nais.io cert, while whatever.ekstern.dev.nav.no uses correct cert