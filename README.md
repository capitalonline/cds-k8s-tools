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

### STEP1：记录当前版本

登录到集群master，记录cds-snat-configuration组件当前版本。应该为`v1.1.2`或`v1.1.3`版本

### STEP2：组件升级

升级cds-snat-configuration到v1.1.4版本

```
kubectl apply -f https://raw.githubusercontent.com/capitalonline/k8s-init/1.19.3/deploy/cds-snat-configuration_old.yaml
```

注意：`/1.19.3/`根据实际集群版本进行调整，当前有：1.20.15、1.19.3、1.17.0

### STEP3：部署状态查询

查看cds-snat-configuration部署状态（所有节点都会部署）

```
$ kubectl get pod -A -owide|grep cds-snat-configuration
kube-system   cds-snat-configuration-28pcw                    1/1     Running   0          3m9s    10.244.4.7   worker001   <none>           <none>
kube-system   cds-snat-configuration-fm9ld                    1/1     Running   0          3m4s    10.244.3.4   worker002   <none>           <none>
kube-system   cds-snat-configuration-vx2fk                    1/1     Running   0          3m15s   10.244.2.7   master003   <none>           <none>
kube-system   cds-snat-configuration-w6sf2                    1/1     Running   0          3m15s   10.244.0.5   master001   <none>           <none>
kube-system   cds-snat-configuration-zs8dr                    1/1     Running   0          3m15s   10.244.1.7   master002   <none>           <none>

$ kubectl get daemonset -n kube-system|grep cds-snat-configuration
cds-snat-configuration   1         1         1       1            1           <none>                   1m


$ kubectl logs cds-snat-configuration-fw29g -n kube-system |head -n 10
INFO[0000] &{10 1 false false 3 60 www.google.com}      
INFO[0000] starting read new ini conf(snat-config) from [/snat/] 
INFO[0000] ended read new ini conf(snat-config) from([/snat/]) 
....
```

### STEP4：异常回滚（部署成功忽略）

若部署失败或者服务异常，将`https://raw.githubusercontent.com/capitalonline/k8s-init/1.19.3/deploy/cds-snat-configuration_old.yaml`文件下载，并将版本改为之前记录的版本

```
image: capitalonline/cds-snat-configuration:v1.1.2
或
image: capitalonline/cds-snat-configuration:v1.1.3
```

后执行

```
kubectl apply -f cds-snat-configuration_old.yaml
```



### STEP5：snat-configmap配置

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

