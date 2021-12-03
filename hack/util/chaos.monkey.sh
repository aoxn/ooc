#!/bin/bash

cat > $CUR_DIR/config.monkey.txt << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: chaosmonkey
  name: chaosmonkey
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: chaosmonkey
  template:
    metadata:
      labels:
        app: chaosmonkey
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: "node-role.kubernetes.io/master"
                    operator: DoesNotExist
      hostNetwork: true
      priorityClassName: system-node-critical
      serviceAccount: admin
      containers:
        - image: ${Registry}/ovm:${Version}
          imagePullPolicy: Always
          name: chaosmonkey-net
          command:
            - /ovm
            - monkey
          volumeMounts:
            - name: bootcfg
              mountPath: /etc/ovm/
              readOnly: true
      tolerations:
        - operator: Exists
      volumes:
        - name: bootcfg
          secret:
            secretName: bootcfg
            items:
              - key: bootcfg
                path: boot.cfg
EOF
