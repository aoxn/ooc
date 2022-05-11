#!/bin/bash

eval < ~/.security/ak.wdrip
echo "use AK config from [~/.security/ak.wdrip] $(cat ~/.security/ak.wdrip )"

echo "use kubeconfig from env KUBECONFIG=[$KUBECONFIG]"

if [[ "$REGION" == "" || "$ACCESS_KEY_ID" == "" || "ACCESS_KEY_SECRET" == "" ]];then
    echo "env REGION ACCESS_KEY_SECRET ACCESS_KEY_ID must not empty" ; exit 1
fi

kubectl --kubeconfig "$KUBECONFIG" apply -f - << EOF
apiVersion: v1
kind: Service
metadata:
  labels:
    app: filebrowser
  name: filebrowser
  namespace: default
spec:
  ports:
    - name: tcp
      port: 80
      protocol: TCP
      targetPort: 80
  selector:
    app: filebrowser
  sessionAffinity: None
  type: LoadBalancer
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: filebrowser
  name: filebrowser
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: filebrowser
  template:
    metadata:
      labels:
        app: filebrowser
    spec:
      priorityClassName: system-node-critical
      containers:
        - image: filebrowser/filebrowser
          imagePullPolicy: Always
          name: filebrowser-net
          command:
            - /filebrowser
            - --database=/srv/database.db
          volumeMounts:
          #  - name: home
          #    mountPath: /srv
            - name: home
              mountPath: /srv
      volumes:
        - name: home
          persistentVolumeClaim:
            claimName: pvc-oss
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-oss
  labels:
    alicloud-pvname: pv-oss
spec:
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  csi:
    driver: ossplugin.csi.alibabacloud.com
    volumeHandle: pv-oss
    nodePublishSecretRef:
      name: oss-secret
      namespace: default
    volumeAttributes:
      bucket: "wdrip-index"
      url: "oss-${REGION}-internal.aliyuncs.com"
      otherOpts: "-o max_stat_cache_size=0 -o allow_other"
      path: "/"
      #authType: "sts"
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-oss
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 5Gi
  selector:
    matchLabels:
      alicloud-pvname: pv-oss
---
apiVersion: v1
kind: Secret
metadata:
  name: oss-secret
  namespace: default
stringData:
  akId: "${ACCESS_KEY_ID}"
  akSecret: "${ACCESS_KEY_SECRET}"

EOF

echo "done"