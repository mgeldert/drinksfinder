# drinksfinder
- The `api_server` directory contains the source code and Dockerfile for the API server.
- The `data_access_layer` directory contains the source code and Dockerfile for the "database".  Note, this is just for PoC purposes; in real life this would have multiple layers including a cache (simulated in the code by some pre-populated tables), an abstraction layer and a back-end database.
- The `terraform` directory contains Terraform code to deploy an AKS cluster in Azure, and adds a namespace and an optional Google Maps API key to the cluster.
- The `kubernetes` directory contains Kubernetes manifests to be applied to the AKS cluster.

To deploy:
- From the `terraform` directory, run `terraform init` and `terraform apply`.  Enter variables when prompted. If you don't have a Google Maps API key, leave this blank but note that the `/api/drinksfinder/v1/pubs/near/postcode/<postcode>` path will be unavailable.
- Once the cluster is deployed, using the `kubeconfig` settings for the new cluster, deploy the manifests in _kubernetes_ (i.e. `kubectl apply -f kubernetes/`)

To access the API:
- Find the endpoint IP address with `kubectl -n drinksfinder get service apiserver -o jsonpath='{.status.loadBalancer.ingress[0].ip}'}`
- Attempt the following requests:
  - `curl http://<IP>/api/drinksfinder/v1/pubs` - list of pubs
  - `curl http://<IP>/api/drinksfinder/v1/pubs?start=3&limit=5` - pagination
  - `curl http://<IP>/api/drinksfinder/v1/pubs?order_by=beer` - sorting (supports "beer", "atmosphere", "amenities" and "value")
  - `curl http://<IP>/api/drinksfinder/v1/pubs?tag=karaoke&tag="free wifi"` - must match all tags
  - `curl http://<IP>/api/drinksfinder/v1/pubs/near` - orders by distance from "primary location" (if you're reading this, you probably know where that is)
  - `curl http://<IP>/api/drinksfinder/v1/pubs/near/postcode/<postcode>` - orders by distance from postcode.  This requires a Google Maps API key to be provided to the `terraform apply`.
  
Pagination and tags work with all searches.  Ordering and location-based searches are mutually exclusive.  If-None-Match/ETag caching is supported.
