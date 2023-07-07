# krakend


## Notes

* Install krakend as a feature in nais-system
* Add default config, no endpoints, consider static documentation portal something
* krakend holds a list of endpoints in a configmap called krakend-partials
* Teams add configmap in each namespace with their own config, we create a watcher for the cms (by annotation/label or something)
* The watcher creates the neccessary endpoints in krakend based on the configmaps

