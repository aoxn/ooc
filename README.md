# WDRIP

## Introduction
`wdrip` is an experimental project for auto-managed distributed kubernetes operating systems which enables you focus on application develop without taking care of underline infrastructure.

Its small and beauty.

Infrastructure agnostic kubernetes cluster management platform with powerful infrastructure resilience ability. High Availability, Easy Recovery

`wdrip` is a kubernetes cluster management operator which API is in alpha state, and might be changed rapidly. You cannot rely on its API stability for now.

**name convention** `wdrip` names from 'water drop' which is the probe ship send by the Three-Body civilization. **Its small but powerful.** 

## GET-STARTED

[**Architecture**](docs/zh/architecture.md)

[**ClusterManager**](docs/zh/manage-cluster.md)

[**Infrastructure Resilience**](docs/zh/infrastructure-resilience.md)

## Build

```shell
# build darwin binary.  and deploy to /usr/local/bin/wdrip
git:(main) ✗ make omac


# build binary and image, release package. need to config your oss ak first.
git:(main) ✗ make build-all 
```

## Attention
- wdrip is a proto-type project for proof of concept, please do not rely on its API stability.

## Contact

Join us in wechat @AoxnKKB

Mail to spacex_nice@163.com
