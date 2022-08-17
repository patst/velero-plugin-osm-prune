# Velero OSM Prune Plugin

This repository is based on the Velero Plugins Example repository: https://github.com/vmware-tanzu/velero-plugin-example/

The OSM Prune plugin removes the containers from [Open Service Mesh](https://github.com/openservicemesh/osm) in order to prevent a duplicate container exception as described at https://github.com/vmware-tanzu/velero/issues/5193 .

## Kinds of Plugin

This is a **Restore Item Action** - performs arbitrary logic on individual items prior to restoring them in the Kubernetes cluster.

For more information, please see the full [plugin documentation](https://velero.io/docs/main/overview-plugins/).

## Building the plugins

To build the plugins, run

```bash
$ make
```

To build the image, run

```bash
$ make container
```

This builds an image tagged as `patst/velero-plugin-osm-prune:main`. If you want to specify a different name or version/tag, run:

```bash
$ IMAGE=your-repo/your-name VERSION=your-version-tag make container 
```

## Deploy Velero test environment

Follow this guide: https://velero.io/docs/v1.9/contributions/minio/

The required YAML files are in the `hack/cluster-config` folder:

``` 
kubectl apply -f ./00-minio-deployment.yaml

velero install \
    --provider aws \
    --plugins velero/velero-plugin-for-aws:v1.2.1 \
    --bucket velero \
    --secret-file ./credentials-velero \
    --use-volume-snapshots=false \
    --use-restic=true \
    --backup-location-config region=minio,s3ForcePathStyle="true",s3Url=http://minio.velero.svc:9000
```

## Deploying the plugins

To deploy your plugin image to an Velero server:

1. Make sure your image is pushed to a registry that is accessible to your cluster's nodes.
2. Run `velero plugin add <registry/image:version>`. Example with a dockerhub image: `velero plugin add patst/velero-plugin-osm-prune:main`.

## Using the plugins

When the plugin is deployed, it is only made available to use. To make the plugin effective, you must modify your configuration:

Backup/Restore actions:

1. Add the plugin to Velero as described in the Deploying the plugins section. (e.g. `velero plugin add patst/velero-plugin-osm-prune:main`)
2. The plugin will be used for the next `backup/restore`.
