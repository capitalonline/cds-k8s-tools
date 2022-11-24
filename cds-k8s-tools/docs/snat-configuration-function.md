# SNat Configuration Function

1. According to the `cds-snat-configuration.yaml`,  we will create/update services such as serviceaccount, clusterrole, clusterrolebinding, configmap, daemonset, etc., where configmap and daemonset are the actual capabilities provided by our services

2. Define the annotation `snat.beta.kubernetes.io/snat-ip` for the nodes (including Master and Worker) that need to be controlled by the SNat Configuration Function in the container cluster

   ```bash
   annotations:
       snat.beta.kubernetes.io/snat-ip: default | snat1 | snat2 | snat3 ....
   ```

   Description：

    -  if annotation `snat.beta.kubernetes.io/snat-ip`  undefined, the node will not be controlled
    -  if annotation `snat.beta.kubernetes.io/snat-ip=`, the node SNat config will point to the `default` group
    -  if annotation `snat.beta.kubernetes.io/snat-ip=default | snat1 | snat2 .....snatn`, the node SNat config will point to the defined group

3. Configure the outbound gateway corresponding to the SNat target group by specifying the key `snat-config` for the configmap `snat-configmap`,  the value of snat-config using the `ini` format

   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: snat-configmap
     namespace: kube-system
   data:
     snat-config: |+
       [default]
       snat.beta.kubernetes.io/default = 10.240.238.4
       snat.beta.kubernetes.io/snat1 = 10.240.238.1
       snat.beta.kubernetes.io/snat2 = 10.240.238.2
       snat.refresh.interval = 60
   ```

   Data description:

   | Key         | Value                                                        | Required | Description                                                  |
      | ----------- | ------------------------------------------------------------ | -------- | ------------------------------------------------------------ |
   | snat-config | [default]<br/># snat.beta.kubernetes.io/default = 10.240.238.4<br/># snat.beta.kubernetes.io/snat1 = 10.240.238.1<br/># snat.beta.kubernetes.io/snat2 = 10.240.238.2<br/>snat.refresh.interval = 60 | yes      | If you want to use the SNat Configuration Function, open the comment and configure it according to the actual.<br/>Every `snat.refresh.interval` time interval, or the value of key `snat-config` changed, pod will check whether the outbound gateway (snat) configuration of the current node is the same as the definition in the current annotiaon, and modify it if not. |
