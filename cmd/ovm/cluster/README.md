
## CreateCluster

**Provider: ROS**
create cluster use ros provider
```
 ./bin/ovm \
    create \
    --name ovm-stack-006 \
    --resource cluster \
    --provider ros \
    --provider-config-file /Users/aoxn/work/ovm/pkg/iaas/provider/ros/example/provider.cfg \
    --boot-config /Users/aoxn/work/ovm/pkg/iaas/provider/ros/example/bootcfg.yaml
```

**watch event**
 ./bin/ovm \
    create \
    --name ovm-stack-006 \
    --resource cluster \
    --provider ros \
    --provider-config-file /Users/aoxn/work/ovm/pkg/iaas/provider/ros/example/provider.cfg \
    --boot-config /Users/aoxn/work/ovm/pkg/iaas/provider/ros/example/bootcfg.yaml
