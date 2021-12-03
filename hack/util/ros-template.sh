#!/usr/bin/env bash

dir=$(cd $(dirname ${BASH_SOURCE});pwd)
ros=${dir}/../../pkg/iaas/provider/ros/

cat ${ros}/demo.json > ${ros}/tpl.go