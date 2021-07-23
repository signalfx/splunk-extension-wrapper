# Local build

To build and package extension to a zip file:

```shell
make
```

The ZIP can now be uploaded as a layer using standard AWS procedure.

# Releasing a layer

After a change is merged to the `master`, please ask Splunk maintainer to release the layer using special GitLab build project.

New ARNs will be published in [the layers repository](https://github.com/signalfx/lambda-layer-versions/blob/master/lambda-extension/lambda-extension-versions.md)