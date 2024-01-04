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



## cds-snat-configuration upgrading v2.0.0

### snat-configmap配置

```
kubectl edit cm snat-configmap -n kube-system
```

### 基础监控配置

| key                | value                                                        |
| ------------------ | ------------------------------------------------------------ |
| snat.check.step    | 检查频率，单位s，默认值为60，最小值为30，最大值为3600，输入必须为整数 |
| snat.check.sum     | 一轮检查总次数：默认值为10，最小值为3，最大值为30，输入值必须为整数 |
| snat.check.limit   | 一轮检查失败次数：默认值为3，最小值为1，最大值为30，不能大于一轮检查总次数，输入值必须为整数 |
| snat.check.recover | 允许异常持续周期：默认值为3                                  |
| snat.check.timeout | 连接超时时长：默认值10，不可修改                             |

### 指标配置

| key                          | value                                                        |
| ---------------------------- | ------------------------------------------------------------ |
| snat.check.node_ping_ext     | 基于node检测ICMP检测地址，输入合法ip/域名，支持批量，采用**英文逗号**`,`分割，默认空 |
| snat.check.node_ping_exclude | 基于node检测ICMP排除地址，输入合法ip/域名，支持批量，采用**英文逗号**`,`分割，默认空 |
| snat.check.pod_telnet_ext    | 基于pod检测TCP检测地址，输入合法ip/域名，支持批量，采用**英文逗号**`,`分割，若存在相同地址的多端口检查，可以采用冒号分割，可查看下面示例，默认空 |
| snat.check.pod_ping_ext      | 基于pod检测ICMP检测地址，输入合法ip/域名，支持批量，采用**英文逗号**`,`分割，默认空 |
| snat.check.pod_ping_default  | pod默认出网能力检查，值：yes/no，默认yes                     |
| snat.check.node_ping_dns     | node默认dns检查，值：yes/no，默认no                          |

**示例**：

```yaml
apiVersion: v1
data:
  snat-config: |+
    [default]
    snat.check.step = 60
    snat.check.sum = 10
    snat.check.limit = 5
    snat.check.recover = 3
    snat.check.timeout = 3
    snat.check.node_ping_ext = 255.255.255.255
    snat.check.node_ping_exclude = 8.8.4.4,8.8.8.8
    snat.check.pod_telnet_ext = www.xxx.com:80:90
    snat.check.pod_ping_ext = 1.1.1.1
    snat.check.pod_ping_default = yes
    snat.check.node_ping_dns = no
```




