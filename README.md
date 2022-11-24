# cds-k8s-tools

Container Tools for Capitalonline(aka. CDS) cloud. This tools allows you to use CDS function with the kubernetes cluster hosted or managed by CDS.

It currently supports CDS SNat Configuration.

The support for other will be added soon.

## To deploy

- To deploy the `SNat Configuration Function` to your k8s, simply run:

  ```bash
  kubectl create -f https://raw.githubusercontent.com/capitalonline/cds-k8s-tools/main/releases/cds-snat-configuration.yaml
  ```

  

## To use

- [SNat Configuration Function](./docs/snat-configuration-function.md)

