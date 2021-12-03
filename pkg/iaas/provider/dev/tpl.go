package dev

var Template = `
{
  "ROSTemplateFormatVersion": "2015-09-01",
  "Description": "Need Activate RAM service",
  "Parameters": {
    "VpcId": {
      "Type": "String",
      "Description": "Set VPC ID, if not provided, create new one.",
      "Label": "VPC ID",
      "Default": "None"
    },
    "VSwitchId": {
      "Type": "String",
      "Description": "Set VSwitch ID, if not provided, create new one",
      "Label": "VSwitch ID",
      "Default": "None"
    },
    "WillReplace": {
      "Type": "Boolean",
      "Description": "Affect existed instance is set true.",
      "Label": "WillReplace",
      "Default": false
    },
    "ContainerCIDR": {
      "Type": "String",
      "Description": "Kubernetes Container CIDR",
      "Label": "Container CIDR",
      "Default": "172.16.0.0/16"
    },
    "ServiceCIDR": {
      "Type": "String",
      "Description": "Kubernetes Service CIDR",
      "Label": "Service CIDR",
      "Default": "172.19.0.0/20"
    },
    "NodeCIDRMask": {
      "Type": "String",
      "Description": "Kubernetes cluster NodeObject CIDR Mask",
      "Label": "NodeObject CIDR Mask",
      "Default": "24"
    },
    "SNatEntry": {
      "Type": "Boolean",
      "Description": "Create SNatEntry. if not, make sure that can access the public network, otherwise the deploy will be fail.",
      "Label": "Create SNatEntry",
      "Default": true
    },
    "NatGateway": {
      "Type": "Boolean",
      "Description": "Create Nat Gateway",
      "Label": "Create Nat Gateway",
      "Default": true
    },
    "NatGatewayId": {
      "Type": "String",
      "Description": "Create a K8S cluster using the existing NatGateway ID",
      "Label": "NatGateway ID",
      "Default": ""
    },
    "SnatTableId": {
      "Type": "String",
      "Description": "Using the existing SnatTable ID",
      "Label": "SnatTable ID",
      "Default": ""
    },
    "EipAddress": {
      "Type": "String",
      "Description": "Attach the existing Eip Address to SnatEntry",
      "Label": "Eip Address",
      "Default": ""
    },
    "Eip": {
      "Type": "Boolean",
      "Description": "Create Eip. if not, make sure that can access the public network, otherwise the deploy will be fail.",
      "Label": "Create Eip",
      "Default": true
    },
    "PublicSLB": {
      "Type": "Boolean",
      "Description": "Create SLB. if not, you can not visit api server outof vpc never.",
      "Label": "Create Public SLB",
      "Default": true
    },
    "MasterImageId": {
      "Type": "String",
      "Description": "ECS Image ID of master nodes",
      "Label": "ECS Image ID",
      "Default": "centos_7"
    },
    "WorkerImageId": {
      "Type": "String",
      "Description": "ECS Image ID of worker nodes",
      "Label": "ECS Image ID",
      "Default": "centos_7"
    },
    "MasterSystemDiskCategory": {
      "Type": "String",
      "Description": "System disk type of master nodes",
      "Label": "System disk type of master nodes",
      "Default": "cloud_ssd"
    },
    "ExecuteVersion": {
      "Type": "Number",
      "Description": "Execute Version to trigger scale",
      "Label": "Execute Version",
      "Default": 0
    },
    "AdjustmentType": {
      "Type": "String",
      "Description": "Ess rule adjustment type",
      "Label": "Ess rule adjustment type",
      "Default": "TotalCapacity"
    },
    "HealthCheckType": {
      "Type": "String",
      "Description": "Ess scaling group health check type",
      "Label": "Ess scaling group health check type",
      "Default": "NONE"
    },
    "ProtectedInstances": {
      "Type": "CommaDelimitedList",
      "Description": "ECS instances of protected mode in the scaling group.",
      "Label": "ECS instances to be protected",
      "Default": ""
    },
    "RemoveInstanceIds": {
      "Type": "CommaDelimitedList",
      "Description": "ECS instances to be deleted from the scaling group.",
      "Label": "ECS instances to be deleted",
      "Default": ""
    },
    "MasterSystemDiskSize": {
      "Type": "Number",
      "Description": "System disk size of master nodes",
      "Label": "System disk size of master nodes",
      "Default": 40
    },
    "MasterDataDisk": {
      "Type": "Boolean",
      "Description": "Whether or not mount a cloud disk for the Master node",
      "Label": "Buy a cloud disk",
      "Default": false
    },
    "MasterDataDiskCategory": {
      "Type": "String",
      "Description": "Disk Category",
      "Label": "Disk Category",
      "Default": "cloud_ssd"
    },
    "MasterDataDiskSize": {
      "Type": "Number",
      "Description": "Disk Size",
      "Label": "Disk Size",
      "Default": 40
    },
    "MasterDataDiskDevice": {
      "Type": "String",
      "Description": "The device where the volume is exposed on the instance",
      "Label": "The device name",
      "Default": "/dev/xvdb"
    },
    "MasterInstanceChargeType": {
      "Type": "String",
      "Description": "Instance charge type: PrePaid or PostPaid",
      "Label": "Billing methods of master node",
      "Default": "PostPaid"
    },
    "MasterPeriod": {
      "Type": "Number",
      "Description": "Period",
      "Default": 3,
      "Label": "Period of master node"
    },
    "MasterPeriodUnit": {
      "Type": "String",
      "Description": "Unit",
      "Default": "Month",
      "Label": "Unit of master node"
    },
    "MasterAutoRenew": {
      "Type": "Boolean",
      "Description": "Whether the prepaid instance will renew automatically after expiration",
      "Default": false,
      "Label": "Auto renew for Prepaid instance"
    },
    "MasterAutoRenewPeriod":{
      "Type": "Number",
      "Description": "The renewal of a single automatic renewal takes a long time. When PeriodUnit=Week the value is {“1”, “2”, “3”};When PeriodUnit=Month the value is {“1”, “2”, “3”, “6”, “12”}",
      "Default": 1,
      "Label": "Auto renew period for Prepaid instance"
    },
    "WorkerSystemDiskCategory": {
      "Type": "String",
      "Description": "System disk type of worker nodes",
      "Label": "System disk type of worker nodes",
      "Default": "cloud_ssd"
    },
    "WorkerSystemDiskSize": {
      "Type": "Number",
      "Description": "System disk size of worker nodes",
      "Label": "System disk size of worker nodes",
      "Default": 40
    },
    "WorkerDataDisk": {
      "Type": "Boolean",
      "Description": "Whether or not mount a cloud disk for the Worker node",
      "Label": "Buy a cloud disk",
      "Default": false
    },
    "WorkerDataDiskCategory": {
      "Type": "String",
      "Description": "Disk Category",
      "Label": "Disk Category",
      "Default": "cloud_ssd"
    },
    "WorkerDataDiskSize": {
      "Type": "Number",
      "Description": "Disk Size",
      "Label": "Disk Size",
      "Default": 40
    },
    "WorkerDataDiskDevice": {
      "Type": "String",
      "Description": "The device where the volume is exposed on the instance",
      "Label": "The device name",
      "Default": "/dev/xvdb"
    },
    "MasterInstanceType": {
      "Type": "String",
      "Description": "Creates ECS instances with the specification for the Master node of Kubernetes",
      "Label": "ECS instance specification of Master node",
      "Default": "ecs.n4.large"
    },
    "WorkerInstanceType": {
      "Type": "String",
      "Description": "Create ESC instances with the specification for the Worker node of Kubernetes",
      "Label": "ECS instance specification of Worker node",
      "Default": "ecs.n4.large"
    },
    "WorkerInstanceChargeType": {
      "Type": "String",
      "Description": "Instance charge type: PrePaid or PostPaid",
      "Label": "Billing methods of worker node",
      "Default": "PostPaid"
    },
    "WorkerPeriod": {
      "Type": "Number",
      "Description": "Period",
      "Default": 3,
      "Label": "Period of worker node"
    },
    "WorkerPeriodUnit": {
      "Type": "String",
      "Description": "Unit",
      "Default": "Month",
      "Label": "Unit of worker node"
    },
    "WorkerAutoRenew": {
      "Type": "Boolean",
      "Description": "Whether the prepaid instance will renew automatically after expiration",
      "Default": false,
      "Label": "Auto renew for Prepaid instance"
    },
    "WorkerAutoRenewPeriod":{
      "Type": "Number",
      "Description": "The renewal of a single automatic renewal takes a long time. When PeriodUnit=Week the value is {“1”, “2”, “3”};When PeriodUnit=Month the value is {“1”, “2”, “3”, “6”, “12”}",
      "Default": 1,
      "Label": "Auto renew period for Prepaid instance"
    },
    "ZoneId": {
      "Type": "String",
      "Default": "",
      "Label": "Zone ID",
      "Description": "Zone ID"
    },
    "GPUFlags": {
      "Type": "Boolean",
      "Default": false,
      "Label": "GPU",
      "Description": "GPU Support"
    },
    "SSHFlags": {
      "Type": "Boolean",
      "Default": false,
      "Label": "Opening of the SSH jumpbox",
      "Description": "Whether to support the opening of the SSH jumpbox"
    },
    "AuditFlags": {
      "Type": "Boolean",
      "Default": true,
      "Label": "Apiserver Auditing",
      "Description": "Whether to open auditing on apiserver"
    },
    "KubernetesVersion": {
      "Type": "String",
      "Description": "Kubernetes Version",
      "Label": "Kubernetes Version",
      "Default": "1.12.6-aliyun.1"
    },
    "DockerVersion": {
      "Type": "String",
      "Description": "Docker Version",
      "Label": "Docker Version",
      "Default": "18.09.2"
    },
    "CloudMonitorVersion": {
      "Type": "String",
      "Description": "Cloud Monitor Agent Version",
      "Label": "Cloud Monitor Version",
      "Default": "1.3.7"
    },
    "EtcdVersion": {
      "Type": "String",
      "Description": "Etcd version",
      "Label": "Etcd version",
      "Default": "v3.3.8"
    },
    "UserCA": {
      "NoEcho": true,
      "Type": "String",
      "Description": "Add user defined CA to cluster trust list",
      "Label": "User CA",
      "Default": "None"
    },
    "Network": {
      "Type": "String",
      "Description": "choose network type for k8s cluster networking",
      "Label": "network type user choose",
      "Default": "None"
    },
    "CA": {
      "NoEcho": true,
      "Type": "String",
      "Description": "Config ca，if not provided, create new one.",
      "Label": "Config CA",
      "Default": "None"
    },
    "Key": {
      "NoEcho": true,
      "Type": "String",
      "Description": "Config CAkey，if not provided, create new one.",
      "Label": "Config CAKey",
      "Default": "None"
    },
    "ClientCA": {
      "NoEcho": true,
      "Type": "String",
      "Description": "Ros Certficate",
      "Label": "Console Certficate",
      "Default": "None"
    },
    "NumOfNodes": {
      "Type": "Number",
      "Description": "Specifies the number of Worker nodes to create Kubernetes",
      "Label": "The number of worker node",
      "Default": "2"
    },
    "CloudMonitorFlags": {
      "Type": "Boolean",
      "Default": false,
      "Label": "Install cloud monitor agent",
      "Description": "If install cloud monitor agent on all nodes"
    },
    "MasterKeyPair": {
      "Type": "String",
      "Description": "Key Pair Name Of master node",
      "Label": "Key Pair Name Of master node",
      "Default": ""
    },
    "WorkerKeyPair": {
      "Type": "String",
      "Description": "Key Pair Name Of worker node",
      "Label": "Key Pair Name Of worker node",
      "Default": ""
    },
    "MasterLoginPassword": {
      "NoEcho": true,
      "Type": "String",
      "Description": "Password Of master node",
      "Label": "Password Of master node",
      "Default": "Just4Test"
    },
    "WorkerLoginPassword": {
      "NoEcho": true,
      "Type": "String",
      "Description": "Password Of worker node",
      "Label": "Password Of worker node",
      "Default": "Just4Test"
    },
    "BetaVersion": {
      "Type": "String",
      "Default": "default",
      "Label": "The version of bate k8s",
      "Description": "This is used for test"
    },
    "ProxyMode": {
      "Type": "String",
      "Description": "The mode we use in kube-proxy.",
      "Label": "The mode we use in kube-proxy.",
      "Default": "iptables"
    },
    "LoggingType": {
      "Type": "String",
      "Description": "Install specified logging collector for k8s cluster",
      "Label": "Logging collector type",
      "Default": "None"
    },
    "SLSProjectName": {
      "Type": "String",
      "Description": "Configure specified sls project name for logging collector",
      "Label": "SLS project name",
      "Default": "None"
    },
    "ElasticSearchHost": {
      "Type": "String",
      "Description": "Configure specified elasticsearch instance host for logging collector",
      "Label": "Elasticsearch instance host",
      "Default": "None"
    },
    "ElasticSearchPort": {
      "Type": "String",
      "Description": "Configure specified elasticsearch instance port for logging collector",
      "Label": "Elasticsearch instance port",
      "Default": "None"
    },
    "ElasticSearchUser": {
      "Type": "String",
      "Description": "Configure specified elasticsearch instance username for logging collector",
      "Label": "Elasticsearch instance username",
      "Default": "None"
    },
    "ElasticSearchPass": {
      "Type": "String",
      "Description": "Configure specified elasticsearch instance password for logging collector",
      "Label": "Elasticsearch instance password",
      "Default": "None"
    },
    "K8SMasterPolicyDocument": {
      "Type": "Json",
      "Description": "A policy document that describes what actions are allowed on which resources , master.",
      "Default": {
        "Version": "1",
        "Statement": [
          {
            "Key": [
              "ecs:Describe*",
              "ecs:AttachDisk",
              "ecs:CreateDisk",
              "ecs:CreateSnapshot",
              "ecs:CreateRouteEntry",
              "ecs:DeleteDisk",
              "ecs:DeleteSnapshot",
              "ecs:DeleteRouteEntry",
              "ecs:DetachDisk",
              "ecs:ModifyAutoSnapshotPolicyEx",
              "ecs:ModifyDiskAttribute",
              "ecs:CreateNetworkInterface",
              "ecs:DescribeNetworkInterfaces",
              "ecs:AttachNetworkInterface",
              "ecs:DetachNetworkInterface",
              "ecs:DeleteNetworkInterface",
              "ecs:DescribeInstanceAttribute"
            ],
            "Resource": [
              "*"
            ],
            "Effect": "Allow"
          },
          {
            "Key": [
              "cr:Get*",
              "cr:List*",
              "cr:PullRepository"
            ],
            "Resource": [
              "*"
            ],
            "Effect": "Allow"
          },
          {
            "Key": [
              "slb:*"
            ],
            "Resource": [
              "*"
            ],
            "Effect": "Allow"
          },
          {
            "Key": [
              "cms:*"
            ],
            "Resource": [
              "*"
            ],
            "Effect": "Allow"
          },
          {
            "Key": [
              "vpc:*"
            ],
            "Resource": [
              "*"
            ],
            "Effect": "Allow"
          },
          {
            "Key": [
              "log:*"
            ],
            "Resource": [
              "*"
            ],
            "Effect": "Allow"
          }
        ]
      }
    },
    "K8sWorkerPolicyDocument": {
      "Type": "Json",
      "Description": "A policy document that describes what actions are allowed on which resources, worker.",
      "Default": {
        "Version": "1",
        "Statement": [
          {
            "Key": [
              "ecs:AttachDisk",
              "ecs:DetachDisk",
              "ecs:DescribeDisks",
              "ecs:CreateDisk",
              "ecs:CreateSnapshot",
              "ecs:DeleteDisk",
              "ecs:CreateNetworkInterface",
              "ecs:DescribeNetworkInterfaces",
              "ecs:AttachNetworkInterface",
              "ecs:DetachNetworkInterface",
              "ecs:DeleteNetworkInterface",
              "ecs:DescribeInstanceAttribute"
            ],
            "Resource": [
              "*"
            ],
            "Effect": "Allow"
          },
          {
            "Key": [
              "cr:Get*",
              "cr:List*",
              "cr:PullRepository"
            ],
            "Resource": [
              "*"
            ],
            "Effect": "Allow"
          },
          {
            "Key": [
              "eci:CreateContainerGroup",
              "eci:DeleteContainerGroup",
              "eci:DescribeContainerGroups",
              "eci:DescribeContainerLog"
            ],
            "Resource": ["*"],
            "Effect": "Allow"
          },
          { "Key": [ "log:*" ], "Resource": [ "*" ], "Effect": "Allow" },
          { "Key": [ "cms:*" ], "Resource": [ "*" ], "Effect": "Allow" },
          { "Key": [ "vpc:*" ], "Resource": [ "*" ], "Effect": "Allow" }
        ]
      }
    }
  },
  "Conditions": {
    "use_ipvs_mode": {
      "Fn::Equals": [
        "ipvs",
        {
          "Ref": "ProxyMode"
        }
      ]
    },
    "create_nat_gateway": {
      "Fn::Equals": [
        true,
        {
          "Ref": "NatGateway"
        }
      ]
    },
    "not_create_nat_gateway": {
      "Fn::Equals": [
        false,
        {
          "Ref": "NatGateway"
        }
      ]
    },
    "create_snat_entry": {
      "Fn::Equals": [
        true,
        {
          "Ref": "SNatEntry"
        }
      ]
    },
    "not_create_snat_entry": {
      "Fn::Equals": [
        false,
        {
          "Ref": "SNatEntry"
        }
      ]
    },
    "create_eip": {
      "Fn::Equals": [
        true,
        {
          "Ref": "Eip"
        }
      ]
    },
    "not_create_eip": {
      "Fn::Equals": [
        false,
        {
          "Ref": "Eip"
        }
      ]
    },
    "create_public_slb": {
      "Fn::Equals": [
        true,
        {
          "Ref": "PublicSLB"
        }
      ]
    },
    "not_create_public_slb": {
      "Fn::Equals": [
        false,
        {
          "Ref": "PublicSLB"
        }
      ]
    },
    "public_slb_ssh_enable": {
      "Fn::And": ["create_public_slb", "ssh_enable"]
    },
    "create_worker_nodes": {
      "Fn::Not": {
        "Fn::Equals": [
          0,
          {
            "Ref": "NumOfNodes"
          }
        ]
      }
    },
    "no_create_worker_nodes": {
      "Fn::Equals": [
        0,
        {
          "Ref": "NumOfNodes"
        }
      ]
    },
    "create_master_data_disk": {
      "Fn::Equals": [
        true,
        {
          "Ref": "MasterDataDisk"
        }
      ]
    },
    "create_worker_data_disk": {
      "Fn::Equals": [
        true,
        {
          "Ref": "WorkerDataDisk"
        }
      ]
    },
    "no_container_cidr": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "ContainerCIDR"
        }
      ]
    },
    "no_service_cidr": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "ServiceCIDR"
        }
      ]
    },
    "create_new_vpc": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "VpcId"
        }
      ]
    },
    "install_cloud_monitor": {
      "Fn::Equals": [
        true,
        {
          "Ref": "CloudMonitorFlags"
        }
      ]
    },
    "user_ca": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "UserCA"
        }
      ]
    },
    "enable_apiserver_audit": {
      "Fn::Equals": [
        true,
        {
          "Ref": "AuditFlags"
        }
      ]
    },
    "network": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "Network"
        }
      ]
    },
    "no_logging_type": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "LoggingType"
        }
      ]
    },
    "no_sls_project_name": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "SLSProjectName"
        }
      ]
    },
    "no_elasticsearch_host": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "ElasticSearchHost"
        }
      ]
    },
    "no_elasticsearch_port": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "ElasticSearchPort"
        }
      ]
    },
    "no_elasticsearch_user": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "ElasticSearchUser"
        }
      ]
    },
    "no_elasticsearch_pass": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "ElasticSearchPass"
        }
      ]
    },
    "cluster_ca": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "CA"
        }
      ]
    },
    "cluster_cakey": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "Key"
        }
      ]
    },
    "client_ca": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "ClientCA"
        }
      ]
    },
    "create_new_vswitch": {
      "Fn::Equals": [
        "None",
        {
          "Ref": "VSwitchId"
        }
      ]
    },
    "gpu-enable": {
      "Fn::Equals": [
        true,
        {
          "Ref": "GPUFlags"
        }
      ]
    },
    "ssh_enable": {
      "Fn::Equals": [
        true,
        {
          "Ref": "SSHFlags"
        }
      ]
    },
    "master_no_password": {
      "Fn::Equals": [
        "",
        {
          "Ref": "MasterLoginPassword"
        }
      ]
    },
    "worker_no_password": {
      "Fn::Equals": [
        "",
        {
          "Ref": "WorkerLoginPassword"
        }
      ]
    },
    "master_no_keypair": {
      "Fn::Equals": [
        "",
        {
          "Ref": "MasterKeyPair"
        }
      ]
    },
    "worker_no_keypair": {
      "Fn::Equals": [
        "",
        {
          "Ref": "WorkerKeyPair"
        }
      ]
    }
  },
  "Resources": {
    "KubernetesMasterRole": {
      "Type": "ALIYUN::RAM::Role",
      "Properties": {
        "RoleName": {
          "Fn::Join": [
            "",
            [
              "KubernetesMasterRole-",
              {
                "Ref": "ALIYUN::StackId"
              }
            ]
          ]
        },
        "Description": "Grant ecs with kubernetes master role.",
        "AssumeRolePolicyDocument": {
          "Statement": [
            {
              "Key": "sts:AssumeRole",
              "Effect": "Allow",
              "Principal": {
                "Service": [
                  "ecs.aliyuncs.com"
                ]
              }
            }
          ],
          "Version": "1"
        },
        "Policies": [
          {
            "PolicyName": {
              "Fn::Join": [
                "",
                [
                  "k8sMasterRolePolicy-",
                  {
                    "Ref": "ALIYUN::StackId"
                  }
                ]
              ]
            },
            "PolicyDocument": {
              "Ref": "K8SMasterPolicyDocument"
            }
          }
        ]
      }
    },
    "KubernetesWorkerRole": {
      "Type": "ALIYUN::RAM::Role",
      "Properties": {
        "RoleName": {
          "Fn::Join": [
            "",
            [
              "KubernetesWorkerRole-",
              {
                "Ref": "ALIYUN::StackId"
              }
            ]
          ]
        },
        "Description": "Grant ecs with kubernetes worker role.",
        "AssumeRolePolicyDocument": {
          "Statement": [
            {
              "Key": "sts:AssumeRole",
              "Effect": "Allow",
              "Principal": {
                "Service": [
                  "ecs.aliyuncs.com"
                ]
              }
            }
          ],
          "Version": "1"
        },
        "Policies": [
          {
            "PolicyName": {
              "Fn::Join": [
                "",
                [
                  "k8sWorkerRolePolicy-",
                  {
                    "Ref": "ALIYUN::StackId"
                  }
                ]
              ]
            },
            "PolicyDocument": {
              "Ref": "K8sWorkerPolicyDocument"
            }
          }
        ]
      }
    },
    "k8s_NAT_Gateway": {
      "Condition": "create_nat_gateway",
      "Type": "ALIYUN::ECS::NatGateway",
      "Properties": {
        "VpcId": {
          "Fn::If": [
            "create_new_vpc",
            {
              "Ref": "k8s_vpc"
            },
            {
              "Ref": "VpcId"
            }
          ]
        },
        "VSwitchId": {
          "Fn::If": [
            "create_new_vswitch",
            {
              "Ref": "k8s_vswitch"
            },
            {
              "Ref": "VSwitchId"
            }
          ]
        }
      }
    },
    "k8s_SNat_Eip": {
      "Condition": "create_eip",
      "Type": "ALIYUN::VPC::EIP",
      "Properties": {
        "InternetChargeType": "PayByTraffic",
        "Bandwidth": "100"
      }
    },
    "k8s_NAT_Gateway_Bind_Eip": {
      "Condition": "create_eip",
      "Type": "ALIYUN::VPC::EIPAssociation",
      "Properties": {
        "InstanceId" : {
          "Fn::If": [
            "create_nat_gateway",
            {
              "Ref": "k8s_NAT_Gateway"
            },
            {
              "Ref": "NatGatewayId"
            }
          ]
        },
        "AllocationId": {
          "Ref": "k8s_SNat_Eip"
        }
      }
    },
    "k8s_vswitch": {
      "Condition": "create_new_vswitch",
      "Type": "ALIYUN::ECS::VSwitch",
      "Properties": {
        "VpcId": {
          "Fn::If": [
            "create_new_vpc",
            {
              "Ref": "k8s_vpc"
            },
            {
              "Ref": "VpcId"
            }
          ]
        },
        "ZoneId": {
          "Ref": "ZoneId"
        },
        "CidrBlock": "192.168.0.0/24"
      }
    },
    "k8s_master_slb_internet": {
      "Condition": "create_public_slb",
      "Type": "ALIYUN::SLB::LoadBalancer",
      "Properties": {
        "LoadBalancerName": "K8sMasterSlbInternet",
        "InternetChargeType": "paybytraffic",
        "LoadBalancerSpec": "slb.s1.small",
        "AddressType": "internet",
        "Tags": [
          {
            "Key": "kubernetes.do.not.delete",
            "Value": {"Ref": "ALIYUN::StackName"}
          }
        ]
      }
    },
    "k8s_master_slb": {
      "Type": "ALIYUN::SLB::LoadBalancer",
      "Properties": {
        "LoadBalancerName": "K8sMasterSlbIntranet",
        "LoadBalancerSpec": "slb.s1.small",
        "VpcId": {
          "Fn::If": [
            "create_new_vpc",
            {
              "Ref": "k8s_vpc"
            },
            {
              "Ref": "VpcId"
            }
          ]
        },
        "VSwitchId": {
          "Fn::If": [
            "create_new_vswitch",
            {
              "Ref": "k8s_vswitch"
            },
            {
              "Ref": "VSwitchId"
            }
          ]
        },
        "AddressType": "intranet",
        "Tags": [
          {
            "Key": "kubernetes.do.not.delete",
            "Value": {"Ref": "ALIYUN::StackName"}
          }
        ]
      }
    },
    "k8s_api_server_listener_internet": {
      "Condition": "create_public_slb",
      "Type": "ALIYUN::SLB::Listener",
      "Properties": {
        "ListenerPort": 6443,
        "BackendServerPort": 6443,
        "LoadBalancerId": {
          "Ref": "k8s_master_slb_internet"
        },
        "Protocol": "tcp",
        "Bandwidth": -1
      }
    },
    "k8s_ssh_listener": {
      "Condition": "public_slb_ssh_enable",
      "Type": "ALIYUN::SLB::Listener",
      "Properties": {
        "ListenerPort": 22,
        "BackendServerPort": 22,
        "LoadBalancerId": {
          "Ref": "k8s_master_slb_internet"
        },
        "Protocol": "tcp",
        "Bandwidth": -1,
        "VServerGroupId": {
          "Ref": "k8s_master_ssh_internet_vgroup"
        }
      }
    },
    "k8s_master_slb_listener_bootstrap": {
      "Type": "ALIYUN::SLB::Listener",
      "Properties": {
        "ListenerPort": 9443,
        "Bandwidth": -1,
        "LoadBalancerId": {
          "Ref": "k8s_master_slb"
        },
        "Protocol": "tcp",
        "BackendServerPort": 32443,
        "VServerGroupId": {
          "Ref": "bootstrap_intranet_vgroup"
        }
      }
    },
    "k8s_master_2": {
      "Type": "ALIYUN::ECS::InstanceGroup",
      "Properties": {
        "WillReplace": {
          "Ref": "WillReplace"
        },
        "IoOptimized": "optimized",
        "InstanceChargeType": {
          "Ref": "MasterInstanceChargeType"
        },
        "Period": {
          "Ref": "MasterPeriod"
        },
        "PeriodUnit": {
          "Ref": "MasterPeriodUnit"
        },
        "AutoRenew": {
          "Ref": "MasterAutoRenew"
        },
        "AutoRenewPeriod": {
          "Ref": "MasterAutoRenewPeriod"
        },
        "VpcId": {
          "Fn::If": [
            "create_new_vpc",
            {
              "Ref": "k8s_vpc"
            },
            {
              "Ref": "VpcId"
            }
          ]
        },
        "DiskMappings": {
          "Fn::If": [
            "create_master_data_disk",
            [{
              "Category": {
                "Ref": "MasterDataDiskCategory"
              },
              "Size": {
                "Ref": "MasterDataDiskSize"
              },
              "Device": {
                "Ref": "MasterDataDiskDevice"
              },
              "DiskName": {
                "Fn::Join": [
                  "-",
                  [
                    {
                      "Ref": "ALIYUN::StackName"
                    },
                    "disk"
                  ]
                ]
              }
            }],
            {
              "Ref":"ALIYUN::NoValue"
            }
          ]
        },
        "RamRoleName": {
          "Fn::GetAtt": [
            "KubernetesMasterRole",
            "RoleName"
          ]
        },
        "UserData": {
          "Fn::Replace": [
            {
              "ros-notify": {
                "Fn::GetAtt": [
                  "k8s_cluster_cloudinit_wait_cond_handle",
                  "CurlCli"
                ]
              }
            },
            {
              "Fn::Join": [
                "",
                [
                  "#!/bin/sh\n",
                  "\n",
                  "set -x\n",
                  "ros-notify\n",
                  "echo notify success immediately\n",
                  "echo 'current node is master2, the bootstrap master'\n",
                  "#############################################################\n",
                  "# This is where parameter started.\n",
                  "# \n",
                  "# \n",
                  "export NAMESPACE=",
                  {
                    "Ref": "BetaVersion"
                  },
                  "\n",
                  "export TOKEN=abcd.efghxxxxxxxx\n",
                  "export OOC_VERSION=0.1.0\n",
                  "export REGION=$(curl --retry 5  -sSL http://100.100.100.200/latest/meta-data/region-id) \n",
                  "export PKG_FILE_SERVER=http://host-oc-${REGION}.oss-${REGION}-internal.aliyuncs.com\n",
                  "export INTRANET_LB=",{"Fn::GetAtt": ["k8s_master_slb", "IpAddress"]}, "\n",
                  "# \n",
                  "# ------------------------------------------------------------\n",
                  "curl --retry 5 -sSL -o /root/run.replace.sh ",
                  "     ${PKG_FILE_SERVER}/ack/${NAMESPACE}/public/run/2.0/run.replace.sh\n",
                  "chmod +x /root/run.replace.sh\n",
                  "ROLE=BOOTSTRAP bash /root/run.replace.sh \n"
                ]
              ]
            }
          ]
        },
        "SecurityGroupId": {
          "Ref": "k8s_sg"
        },
        "VSwitchId": {
          "Fn::If": [
            "create_new_vswitch",
            {
              "Ref": "k8s_vswitch"
            },
            {
              "Ref": "VSwitchId"
            }
          ]
        },
        "ImageId": {
          "Ref": "MasterImageId"
        },
        "AllocatePublicIP": false,
        "InstanceType": {
          "Ref": "MasterInstanceType"
        },
        "InstanceName": {
          "Fn::Join": [
            "-",
            [
              "master-02",
              {
                "Ref": "ALIYUN::StackName"
              }
            ]
          ]
        },
        "MaxAmount": "1",
        "MinAmount":"1",
        "SystemDiskSize": {
          "Ref": "MasterSystemDiskSize"
        },
        "SystemDiskCategory": {
          "Ref": "MasterSystemDiskCategory"
        },
        "Password": {
          "Fn::If": [
            "master_no_password",
            {
              "Ref": "ALIYUN::NoValue"
            },
            {
              "Ref": "MasterLoginPassword"
            }
          ]
        },
        "KeyPairName": {
          "Fn::If": [
            "master_no_keypair",
            {
              "Ref": "ALIYUN::NoValue"
            },
            {
              "Ref": "MasterKeyPair"
            }
          ]
        }
      }
    },
    "k8s_master_slb_listener": {
      "Type": "ALIYUN::SLB::Listener",
      "Properties": {
        "ListenerPort": 6443,
        "Bandwidth": -1,
        "LoadBalancerId": {
          "Ref": "k8s_master_slb"
        },
        "Protocol": "tcp",
        "BackendServerPort": 6443
      }
    },
    "k8s_master_3": {
      "Type": "ALIYUN::ECS::InstanceGroup",
      "Properties": {
        "WillReplace": {
          "Ref": "WillReplace"
        },
        "IoOptimized": "optimized",
        "InstanceChargeType": {
          "Ref": "MasterInstanceChargeType"
        },
        "Period": {
          "Ref": "MasterPeriod"
        },
        "PeriodUnit": {
          "Ref": "MasterPeriodUnit"
        },
        "AutoRenew": {
          "Ref": "MasterAutoRenew"
        },
        "AutoRenewPeriod": {
          "Ref": "MasterAutoRenewPeriod"
        },
        "RamRoleName": {
          "Fn::GetAtt": [
            "KubernetesMasterRole",
            "RoleName"
          ]
        },
        "VpcId": {
          "Fn::If": [
            "create_new_vpc",
            {
              "Ref": "k8s_vpc"
            },
            {
              "Ref": "VpcId"
            }
          ]
        },
        "DiskMappings": {
          "Fn::If": [
            "create_master_data_disk",
            [{
              "Category": {
                "Ref": "MasterDataDiskCategory"
              },
              "Size": {
                "Ref": "MasterDataDiskSize"
              },
              "Device": {
                "Ref": "MasterDataDiskDevice"
              },
              "DiskName": {
                "Fn::Join": [
                  "-",
                  [
                    {
                      "Ref": "ALIYUN::StackName"
                    },
                    "disk"
                  ]
                ]
              }
            }],
            {
              "Ref":"ALIYUN::NoValue"
            }
          ]
        },
        "UserData": {
          "Fn::Replace": [
            {
              "ros-notify": {
                "Fn::GetAtt": [
                  "k8s_cluster_cloudinit_wait_cond_handle",
                  "CurlCli"
                ]
              }
            },
            {
              "Fn::Join": [
                "",
                [
                  "#!/bin/sh\n",
                  "\n",
                  "set -x\n",
                  "echo 'current node is master3, the join master'\n",
                  "#############################################################\n",
                  "# This is where parameter started.\n",
                  "# \n",
                  "export NAMESPACE=",{"Ref": "BetaVersion"},"\n",
                  "export TOKEN=abcd.efghxxxxxxxx\n",
                  "export OOC_VERSION=0.1.0\n",
                  "export REGION=$(curl --retry 5  -sSL http://100.100.100.200/latest/meta-data/region-id) \n",
                  "export PKG_FILE_SERVER=http://host-oc-${REGION}.oss-${REGION}-internal.aliyuncs.com\n",
                  "export ENDPOINT=",{"Fn::Join": ["", ["http://", {"Fn::GetAtt": ["k8s_master_slb", "IpAddress"]}, ":9443"]]}, "\n",
                  "# \n",
                  "# ------------------------------------------------------------\n",
                  "curl --retry 5 -sSL -o /root/run.replace.sh ",
                  "     ${PKG_FILE_SERVER}/ack/${NAMESPACE}/public/run/2.0/run.replace.sh\n",
                  "chmod +x /root/run.replace.sh\n",

                  "ROLE=MASTER bash /root/run.replace.sh \n",

                  "ros-notify\n"
                ]
              ]
            }
          ]
        },
        "SecurityGroupId": {
          "Ref": "k8s_sg"
        },
        "VSwitchId": {
          "Fn::If": [
            "create_new_vswitch",
            {
              "Ref": "k8s_vswitch"
            },
            {
              "Ref": "VSwitchId"
            }
          ]
        },
        "ImageId": {
          "Ref": "MasterImageId"
        },
        "AllocatePublicIP": false,
        "InstanceType": {
          "Ref": "MasterInstanceType"
        },
        "InstanceName": {
          "Fn::Join": [
            "-",
            [
              "master-03",
              {
                "Ref": "ALIYUN::StackName"
              }
            ]
          ]
        },
        "MaxAmount": "1",
        "MinAmount":"1",
        "SystemDiskSize": {
          "Ref": "MasterSystemDiskSize"
        },
        "SystemDiskCategory": {
          "Ref": "MasterSystemDiskCategory"
        },
        "Password": {
          "Fn::If": [
            "master_no_password",
            {
              "Ref": "ALIYUN::NoValue"
            },
            {
              "Ref": "MasterLoginPassword"
            }
          ]
        },
        "KeyPairName": {
          "Fn::If": [
            "master_no_keypair",
            {
              "Ref": "ALIYUN::NoValue"
            },
            {
              "Ref": "MasterKeyPair"
            }
          ]
        }
      }
    },
    "k8s_master_1": {
      "Type": "ALIYUN::ECS::InstanceGroup",
      "Condition": "create_snat_entry",
      "Properties": {
        "WillReplace": {
          "Ref": "WillReplace"
        },
        "IoOptimized": "optimized",
        "InstanceChargeType": {
          "Ref": "MasterInstanceChargeType"
        },
        "Period": {
          "Ref": "MasterPeriod"
        },
        "PeriodUnit": {
          "Ref": "MasterPeriodUnit"
        },
        "AutoRenew": {
          "Ref": "MasterAutoRenew"
        },
        "AutoRenewPeriod": {
          "Ref": "MasterAutoRenewPeriod"
        },
        "RamRoleName": {
          "Fn::GetAtt": [
            "KubernetesMasterRole",
            "RoleName"
          ]
        },
        "VpcId": {
          "Fn::If": [
            "create_new_vpc",
            {
              "Ref": "k8s_vpc"
            },
            {
              "Ref": "VpcId"
            }
          ]
        },
        "DiskMappings": {
          "Fn::If": [
            "create_master_data_disk",
            [{
              "Category": {
                "Ref": "MasterDataDiskCategory"
              },
              "Size": {
                "Ref": "MasterDataDiskSize"
              },
              "Device": {
                "Ref": "MasterDataDiskDevice"
              },
              "DiskName": {
                "Fn::Join": [
                  "-",
                  [
                    {
                      "Ref": "ALIYUN::StackName"
                    },
                    "disk"
                  ]
                ]
              }
            }],
            {
              "Ref":"ALIYUN::NoValue"
            }
          ]
        },
        "UserData": {
          "Fn::Replace": [
            {
              "ros-notify": {
                "Fn::GetAtt": [
                  "k8s_cluster_cloudinit_wait_cond_handle",
                  "CurlCli"
                ]
              }
            },
            {
              "Fn::Join": [
                "",
                [
                  "#!/bin/sh\n",
                  "\n",
                  "set -x\n",
                  "echo 'current node is master1, the join master'\n",
                  "#############################################################\n",
                  "# This is where parameter started.\n",
                  "# \n",
                  "export NAMESPACE=",{"Ref": "BetaVersion"},"\n",
                  "export TOKEN=abcd.efghxxxxxxxx\n",
                  "export OOC_VERSION=0.1.0\n",
                  "export REGION=$(curl --retry 5 -sSL http://100.100.100.200/latest/meta-data/region-id) \n",
                  "export PKG_FILE_SERVER=http://host-oc-${REGION}.oss-${REGION}-internal.aliyuncs.com\n",
                  "export ENDPOINT=",{"Fn::Join": ["", ["http://", {"Fn::GetAtt": ["k8s_master_slb", "IpAddress"]}, ":9443"]]}, "\n",
                  "# \n",
                  "# ------------------------------------------------------------\n",
                  "curl --retry 5 -sSL -o /root/run.replace.sh ",
                  "     ${PKG_FILE_SERVER}/ack/${NAMESPACE}/public/run/2.0/run.replace.sh\n",
                  "chmod +x /root/run.replace.sh\n",

                  "ROLE=MASTER bash /root/run.replace.sh \n",

                  "ros-notify\n"
                ]
              ]
            }
          ]
        },
        "SecurityGroupId": {
          "Ref": "k8s_sg"
        },
        "VSwitchId": {
          "Fn::If": [
            "create_new_vswitch",
            {
              "Ref": "k8s_vswitch"
            },
            {
              "Ref": "VSwitchId"
            }
          ]
        },
        "ImageId": {
          "Ref": "MasterImageId"
        },
        "AllocatePublicIP": false,
        "InstanceType": {
          "Ref": "MasterInstanceType"
        },
        "InstanceName": {
          "Fn::Join": [
            "-",
            [
              "master-01",
              {
                "Ref": "ALIYUN::StackName"
              }
            ]
          ]
        },
        "MaxAmount": "1",
        "MinAmount":"1",
        "SystemDiskSize": {
          "Ref": "MasterSystemDiskSize"
        },
        "SystemDiskCategory": {
          "Ref": "MasterSystemDiskCategory"
        },
        "Password": {
          "Fn::If": [
            "master_no_password",
            {
              "Ref": "ALIYUN::NoValue"
            },
            {
              "Ref": "MasterLoginPassword"
            }
          ]
        },
        "KeyPairName": {
          "Fn::If": [
            "master_no_keypair",
            {
              "Ref": "ALIYUN::NoValue"
            },
            {
              "Ref": "MasterKeyPair"
            }
          ]
        }
      }
    },
    "k8s_vpc": {
      "Condition": "create_new_vpc",
      "Type": "ALIYUN::ECS::VPC",
      "Properties": {
        "CidrBlock": "192.168.0.0/16",
        "VpcName": {
          "Fn::Join": [
            "-",
            [
              "vpc",
              {
                "Ref": "ALIYUN::StackName"
              }
            ]
          ]
        }
      }
    },
    "k8s_master_slb_attachements": {
      "Type": "ALIYUN::SLB::BackendServerAttachment",
      "Properties": {
        "BackendServerList": [
          {
            "Ref": "k8s_master_1"
          },
          {
            "Ref": "k8s_master_2"
          },
          {
            "Ref": "k8s_master_3"
          }
        ],
        "LoadBalancerId": {
          "Ref": "k8s_master_slb"
        }
      }
    },
    "k8s_master_slb_attachements_bootstrap": {
      "Type": "ALIYUN::SLB::BackendServerAttachment",
      "Properties": {
        "BackendServerList": [
          {
            "Ref": "k8s_master_2"
          }
        ],
        "LoadBalancerId": {
          "Ref": "k8s_master_slb"
        }
      }
    },
    "k8s_master_slb_internet_attachements": {
      "Condition": "create_public_slb",
      "Type": "ALIYUN::SLB::BackendServerAttachment",
      "Properties": {
        "BackendServerList": [
          {
            "Ref": "k8s_master_1"
          },
          {
            "Ref": "k8s_master_2"
          },
          {
            "Ref": "k8s_master_3"
          }
        ],
        "LoadBalancerId": {
          "Ref": "k8s_master_slb_internet"
        }
      }
    },
    "k8s_master_ssh_internet_vgroup": {
      "Condition": "create_public_slb",
      "Type": "ALIYUN::SLB::VServerGroup",
      "Properties": {
        "VServerGroupName": "sshVirtualGroup",
        "BackendServers": [
          {
            "ServerId": {
              "Ref": "k8s_master_3"
            },
            "Weight": 100,
            "Port": 22
          }
        ],
        "LoadBalancerId": {
          "Ref": "k8s_master_slb_internet"
        }
      }
    },
    "bootstrap_intranet_vgroup": {
      "Type": "ALIYUN::SLB::VServerGroup",
      "Properties": {
        "VServerGroupName": "bootstrap9443",
        "BackendServers": [
          {
            "ServerId": {
              "Ref": "k8s_master_2"
            },
            "Weight": 100,
            "Port": 32443
          }
        ],
        "LoadBalancerId": {
          "Ref": "k8s_master_slb"
        }
      }
    },
    "k8s_cluster_cloudinit_wait_cond_handle": {
      "Type": "ALIYUN::ROS::WaitConditionHandle"
    },
    "k8s_sg": {
      "Type": "ALIYUN::ECS::SecurityGroup",
      "Properties": {
        "Description" : "This is used by kubernetes. Do not delete please!",
        "Tags": [
          {
            "Key": "kubernetes.do.not.delete",
            "Value": {"Ref": "ALIYUN::StackName"}
          }
        ],
        "VpcId": {
          "Fn::If": [
            "create_new_vpc",
            {
              "Ref": "k8s_vpc"
            },
            {
              "Ref": "VpcId"
            }
          ]
        },
        "SecurityGroupName": "k8s_sg",
        "SecurityGroupIngress": [
          {
            "Description" : "This is used by kubernetes. Do not delete please!",
            "PortRange": "-1/-1",
            "Priority": 1,
            "SourceCidrIp": {
              "Ref": "ContainerCIDR"
            },
            "IpProtocol": "all",
            "NicType": "intranet"
          },
          {
            "Description" : "This is used by kubernetes. Do not delete please!",
            "SourceCidrIp": "0.0.0.0/0",
            "IpProtocol": "icmp",
            "NicType": "intranet",
            "Policy": "accept",
            "PortRange": "-1/-1",
            "Priority": 1
          }
        ],
        "SecurityGroupEgress": [
          {
            "Description" : "This is used by kubernetes. Do not delete please!",
            "PortRange": "-1/-1",
            "Priority": 1,
            "IpProtocol": "all",
            "DestCidrIp": "0.0.0.0/0",
            "NicType": "intranet"
          }
        ]
      }
    },
    "k8s_NAT_Gateway_SNATEntry": {
      "Condition": "create_snat_entry",
      "Type": "ALIYUN::ECS::SNatEntry",
      "Properties": {
        "SourceVSwitchId": {
          "Fn::If": [
            "create_new_vswitch",
            {
              "Ref": "k8s_vswitch"
            },
            {
              "Ref": "VSwitchId"
            }
          ]
        },
        "SNatTableId": {
          "Fn::If": [
            "create_nat_gateway",
            {
              "Fn::GetAtt": [
                "k8s_NAT_Gateway",
                "SNatTableId"
              ]
            },
            {
              "Ref": "SnatTableId"
            }
          ]
        },
        "SNatIp": {
          "Fn::If": [
            "create_eip",
            {
              "Fn::GetAtt": [
                "k8s_NAT_Gateway_Bind_Eip",
                "EipAddress"
              ]
            },
            {
              "Ref": "EipAddress"
            }
          ]
        }
      }
    },
    "k8s_cluster_cloudinit_wait_cond": {
      "Type": "ALIYUN::ROS::WaitCondition",
      "Properties": {
        "Timeout": 1800,
        "Count": {"Fn::Add": [{"Ref": "NumOfNodes"}, 2]},
        "Handle": {
          "Ref": "k8s_cluster_cloudinit_wait_cond_handle"
        }
      }
    },
    "k8s_nodes_sg": {
      "Type": "ALIYUN::ESS::ScalingGroup",
      "Properties": {
        "MinSize": "0",
        "MaxSize": "1000",
        "DefaultCooldown": 0,
        "ScalingGroupName": {
          "Fn::Join": [
            "-",
            [
              "k8s",
              {
                "Ref": "ALIYUN::StackId"
              }
            ]
          ]
        },
        "VSwitchId": {
          "Fn::If": [
            "create_new_vswitch",
            {
              "Ref": "k8s_vswitch"
            },
            {
              "Ref": "VSwitchId"
            }
          ]
        },
        "MultiAZPolicy": "BALANCE",
        "ProtectedInstances": {
          "Ref": "ProtectedInstances"
        },
        "RemovalPolicys": ["OldestScalingConfiguration", "NewestInstance"],
        "HealthCheckType": {
          "Ref": "HealthCheckType"
        }
      }
    },
    "k8s_nodes_config": {
      "Type": "ALIYUN::ESS::ScalingConfiguration",
      "Properties": {
        "ScalingGroupId": {
          "Ref": "k8s_nodes_sg"
        },
        "IoOptimized": "optimized",
        "InstanceChargeType": {
          "Ref": "WorkerInstanceChargeType"
        },
        "Period": {
          "Ref": "WorkerPeriod"
        },
        "PeriodUnit": {
          "Ref": "WorkerPeriodUnit"
        },
        "AutoRenew": {
          "Ref": "WorkerAutoRenew"
        },
        "AutoRenewPeriod": {
          "Ref": "WorkerAutoRenewPeriod"
        },
        "DiskMappings": {
          "Fn::If": [
            "create_worker_data_disk",
            [{
              "Category": {
                "Ref": "WorkerDataDiskCategory"
              },
              "Size": {
                "Ref": "WorkerDataDiskSize"
              },
              "Device": {
                "Ref": "WorkerDataDiskDevice"
              }
            }],
            {
              "Ref":"ALIYUN::NoValue"
            }
          ]
        },
        "RamRoleName": {
          "Fn::GetAtt": [
            "KubernetesWorkerRole",
            "RoleName"
          ]
        },
        "UserData": {
          "Fn::Replace": [
            {
              "ros-notify": {
                "Fn::GetAtt": [
                  "k8s_cluster_cloudinit_wait_cond_handle",
                  "CurlCli"
                ]
              }
            },
            {
              "Fn::Join": [
                "",
                [
                  "#!/bin/sh\n",
                  "\n",
                  "set -x\n",
                  "echo 'current node is workernodes, the join nodes'\n",
                  "#############################################################\n",
                  "# This is where parameter started.\n",
                  "# \n",
                  "# \n",
                  "export NAMESPACE=",{"Ref": "BetaVersion"},"\n",
                  "export TOKEN=abcd.efghxxxxxxxx\n",
                  "export OOC_VERSION=0.1.0\n",
                  "export REGION=$(curl --retry 5 -sSL http://100.100.100.200/latest/meta-data/region-id) \n",
                  "export PKG_FILE_SERVER=http://host-oc-${REGION}.oss-${REGION}-internal.aliyuncs.com\n",
                  "export ENDPOINT=",{"Fn::Join": ["", ["http://", {"Fn::GetAtt": ["k8s_master_slb", "IpAddress"]}, ":9443"]]}, "\n",
                  "# \n",
                  "# ------------------------------------------------------------\n",
                  "curl --retry 5 -sSL -o /root/run.replace.sh ",
                  "     ${PKG_FILE_SERVER}/ack/${NAMESPACE}/public/run/2.0/run.replace.sh\n",
                  "chmod +x /root/run.replace.sh\n",

                  "ROLE=WORKER bash /root/run.replace.sh\n",

                  "ros-notify\n"
                ]
              ]
            }
          ]
        },
        "SecurityGroupId": {
          "Ref": "k8s_sg"
        },
        "ImageId": {
          "Ref": "WorkerImageId"
        },
        "InstanceTypes": [{
          "Ref": "WorkerInstanceType"
        }],
        "SystemDiskSize": {
          "Ref": "WorkerSystemDiskSize"
        },
        "SystemDiskCategory": {
          "Ref": "WorkerSystemDiskCategory"
        },
        "InstanceName": {
          "Fn::Join": [
            "-",
            [
              "worker",
              {
                "Ref": "ALIYUN::StackName"
              }
            ]
          ]
        },
        "Password": {
          "Fn::If": [
            "worker_no_password",
            {
              "Ref": "ALIYUN::NoValue"
            },
            {
              "Ref": "WorkerLoginPassword"
            }
          ]
        },
        "KeyPairName": {
          "Fn::If": [
            "worker_no_keypair",
            {
              "Ref": "ALIYUN::NoValue"
            },
            {
              "Ref": "WorkerKeyPair"
            }
          ]
        }
      }
    },
    "k8s_nodes_scaling_rule": {
      "Type": "ALIYUN::ESS::ScalingRule",
      "Condition": "create_worker_nodes",
      "Properties": {
        "AdjustmentType": {
          "Ref": "AdjustmentType"
        },
        "ScalingGroupId": {
          "Ref": "k8s_nodes_sg"
        },
        "AdjustmentValue": {
          "Ref": "NumOfNodes"
        }
      }
    },
    "k8s_nodes": {
      "Type": "ALIYUN::ESS::ScalingGroupEnable",
      "Condition": "create_worker_nodes",
      "Properties": {
        "ScalingGroupId": {
          "Ref": "k8s_nodes_sg"
        },
        "ScalingConfigurationId": {
          "Ref": "k8s_nodes_config"
        },
        "ScalingRuleAris": [
          {
            "Fn::GetAtt": [
              "k8s_nodes_scaling_rule",
              "ScalingRuleAri"
            ]
          }
        ],
        "ScalingRuleArisExecuteVersion": {
          "Ref": "ExecuteVersion"
        },
        "RemoveInstanceIds": {
          "Ref": "RemoveInstanceIds"
        }
      }
    }
  },
  "Outputs": {
    "APIServerIntranet": {
      "Value": {
        "Fn::Join": [
          "",
          [
            "https://",
            {
              "Fn::GetAtt": [
                "k8s_master_slb",
                "IpAddress"
              ]
            },
            ":6443"
          ]
        ]
      },
      "Description": "API Server Ros IP"
    },
    "APIServerInternet": {
      "Description": "API Server Public IP",
      "Value": {
        "Fn::If": [
          "create_public_slb",
          {
            "Fn::Join": [
              "",
              [
                "https://",
                {
                  "Fn::GetAtt": [
                    "k8s_master_slb_internet",
                    "IpAddress"
                  ]
                },
                ":6443"
              ]
            ]
          },
          ""
        ]

      }
    },
    "JumpHost": {
      "Description": "SSH login on the node by this ip",
      "Value": {
        "Fn::If": [
          "create_public_slb",
          {
            "Fn::GetAtt": [
              "k8s_master_slb_internet",
              "IpAddress"
            ]
          },
          ""
        ]

      }
    },
    "MasterIPs": {
      "Value": [
        { "Fn::Select" : [
          "0",
          {
            "Fn::GetAtt": [
              "k8s_master_1",
              "PrivateIps"
            ]
          }
        ]},
        {
          "Fn::Select" : [
            "0",
            {
              "Fn::GetAtt": [
                "k8s_master_2",
                "PrivateIps"
              ]
            }
          ]
        },
        {
          "Fn::Select" : [
            "0",
            {
              "Fn::GetAtt": [
                "k8s_master_3",
                "PrivateIps"
              ]
            }
          ]
        }
      ],
      "Description": "Private IP Master node"
    },
    "MasterInstanceIDs": {
      "Value": [
        { "Fn::Select" : [
          "0",
          {
            "Fn::GetAtt": [
              "k8s_master_1",
              "PrivateIps"
            ]
          }
        ]},
        {
          "Fn::Select" : [
            "0",
            {
              "Fn::GetAtt": [
                "k8s_master_2",
                "InstanceIds"
              ]
            }
          ]
        },
        {
          "Fn::Select" : [
            "0",
            {
              "Fn::GetAtt": [
                "k8s_master_3",
                "InstanceIds"
              ]
            }
          ]
        }
      ],
      "Description": "Ids of master node"
    },
    "NodeInstanceIDs": {
      "Value": {
        "Fn::If": [
          "create_worker_nodes",
          {
            "Fn::GetAtt": [
              "k8s_nodes",
              "ScalingInstances"
            ]
          },
          ""
        ]
      },
      "Description": "Ids of worker node"
    },
    "NodesScalingAddedInstances": {
      "Description": "Count of ess scaling instance",
      "Value": {
        "Fn::If": [
          "create_worker_nodes",
          {
            "Fn::GetAtt": [
              "k8s_nodes",
              "ScalingRuleArisExecuteResultNumberOfAddedInstances"
            ]
          },
          ""
        ]
      }
    },
    "NodesScalingErrorInfo": {
      "Description": "Error msg of ess scaling instance",
      "Value": {
        "Fn::If": [
          "create_worker_nodes",
          {
            "Fn::GetAtt": [
              "k8s_nodes",
              "ScalingRuleArisExecuteErrorInfo"
            ]
          },
          ""
        ]
      }
    },
    "VpcId": {
      "Value": {
        "Fn::If": [
          "create_new_vpc",
          {
            "Ref": "k8s_vpc"
          },
          {
            "Ref": "VpcId"
          }
        ]
      },
      "Description": "VPC ID"
    },
    "VSwitchId": {
      "Value": {
        "Fn::If": [
          "create_new_vswitch",
          {
            "Ref": "k8s_vswitch"
          },
          {
            "Ref": "VSwitchId"
          }
        ]
      },
      "Description": "Vswitch ID"
    },
    "NatGatewayId": {
      "Description": "Nat gateway ID",
      "Value": {
        "Fn::If": [
          "create_nat_gateway",
          {
            "Fn::GetAtt": [
              "k8s_NAT_Gateway",
              "NatGatewayId"
            ]
          },
          {
            "Ref": "NatGatewayId"
          }
        ]
      }
    },
    "InternetSlbId": {
      "Description": "Internet SLB ID",
      "Value": {
        "Fn::If": [
          "create_public_slb",
          {
            "Fn::GetAtt": [
              "k8s_master_slb_internet",
              "LoadBalancerId"
            ]
          },
          ""
        ]
      }
    },
    "IntranetSlbId": {
      "Description": "Intranet SLB ID",
      "Value": {
        "Fn::GetAtt": [
          "k8s_master_slb",
          "LoadBalancerId"
        ]
      }
    },
    "ProxyMode": {
      "Description": "The mode we use in kube-proxy.",
      "Value": {
        "Ref": "ProxyMode"
      }
    },
    "LastKnownError": {
      "Description": "Log Info Output",
      "Value":
      {
        "Fn::Join":[
          "",
          [
            {"Fn::GetJsonValue": ["errmsg", { "Fn::GetAtt": ["k8s_cluster_cloudinit_wait_cond", "Data"]}]},
          ]
        ]
      }
    }
  }
}
`
