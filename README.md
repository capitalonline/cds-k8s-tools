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



## cds-snat-configuration upgrading v1.1.4 

### snat-configmap配置

编辑`snat-configmap`，并保存

```
kubectl edit cm snat-configmap -n kube-system
```

此版本configmap增加四个字段，分别是

- `snat.check.pod_ping_ext`：检测pod出网能力需要ping的ip/域名
- `snat.check.pod_ping_exclude`：检测pod出网能力需要过滤掉ping的ip/域名
- `snat.check.node_ping_ext`：检测node出网能力需要ping的ip/域名
- `snat.check.node_ping_exclude`：检测node出网能力需要过滤掉ping的ip/域名

**注意：**

1. 支持批量，采用**英文逗号**分割`,`        
2.  若不需要相应配置，直接**注释**或者**不写**即可，示例如下：

```
snat.check.pod_ping_ext = www.163.com,www.baidu.com
# snat.check.pod_ping_exclude = www.163.com
# snat.check.node_ping_ext = 202.103.0.117
snat.check.node_ping_exclude = 8.8.4.4,8.8.8.8
```

**下面列出常用配置场景：**

#### 场景一：过滤 ping dns 8.8.4.4

在data内容中，新增一个字段`snat.check.node_ping_exclude = 8.8.4.4`，示例如下：

```
data:
  snat-config: |
    [default]
    ...
    ...
    ...
    snat.check.node_ping_exclude = 8.8.4.4
```

#### 场景二：pod增加更多需要ping的域名

```
data:
  snat-config: |
    [default]
    ...
    ...
    ...
   snat.check.pod_ping_ext = www.163.com,www.baidu.com,www.google.com
```

