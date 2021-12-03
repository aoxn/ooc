## Config Contents

Fixed Config
```
iaas:
  os:
    image: abclid.vxd
    disk:
      size: 40G
      type: cloudssd
    secret:
      type: password|secret
      value: 
        name: username| keyname
        pass: password
    kernel:
      sysctl:
      - abc
      - mcd
cluster:
  cloud: public
  kubernetes:
    name: kubernetes
    version: 1.12.6-aliyun.1
  etcd:
    name: etcd
    version: 3.3.8    
  docker:
    name: docker
    version: 18.09.2
    para: 
      key1: value
      key2: value2
  betaversion: ""
  sans:
  - 192.168.0.1
  network:
    mode: ipvs
    podcidr: 192.168.0.1/16
    svccidr: 172.10.10.2/20
    domain: cluster.domain
  endpoints:
    intranet: 192.168.0.1
    internet: 11.1.1.1
  immutable:
    ca:
      root:
        key: aaa
        value: bbb
      front: 
        key: aaa
        value: bbb
    
```

ca:

immutable
    ipvs