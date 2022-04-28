package alibaba

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
      "Description": "Kubernetes cluster Node CIDR Mask",
      "Label": "Node CIDR Mask",
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
      "Default": "centos_7_7_x64_20G_alibase_20200426.vhd"
    },
    "MasterSystemDiskCategory": {
      "Type": "String",
      "Description": "System disk type of master nodes",
      "Label": "System disk type of master nodes",
      "Default": "cloud_essd"
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
      "Default": "cloud_essd"
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
    "MasterInstanceType": {
      "Type": "String",
      "Description": "Creates ECS instances with the specification for the Master node of Kubernetes",
      "Label": "ECS instance specification of Master node",
      "Default": "ecs.c5.xlarge"
    },
    "ZoneId": {
      "Type": "String",
      "Default": "cn-hangzhou-h",
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
      "Description": "Cluster ca，if not provided, create new one.",
      "Label": "Cluster CA",
      "Default": "None"
    },
    "Key": {
      "NoEcho": true,
      "Type": "String",
      "Description": "Cluster CAkey，if not provided, create new one.",
      "Label": "Cluster CAKey",
      "Default": "None"
    },
    "ClientCA": {
      "NoEcho": true,
      "Type": "String",
      "Description": "Client Certficate",
      "Label": "Console Certficate",
      "Default": "None"
    },
    "CloudMonitorFlags": {
      "Type": "Boolean",
      "Default": false,
      "Label": "Install cloud monit agent",
      "Description": "If install cloud monit agent on all nodes"
    },
    "MasterKeyPair": {
      "Type": "String",
      "Description": "Key Pair Name Of master node",
      "Label": "Key Pair Name Of master node",
      "Default": ""
    },
    "MasterLoginPassword": {
      "NoEcho": true,
      "Type": "String",
      "Description": "Password Of master node",
      "Label": "Password Of master node",
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
            "Action": [
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
              "ecs:AssignPrivateIpAddresses",
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
            "Action": [
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
            "Action": [
              "slb:*"
            ],
            "Resource": [
              "*"
            ],
            "Effect": "Allow"
          },
          {
            "Action": [
              "cms:*"
            ],
            "Resource": [
              "*"
            ],
            "Effect": "Allow"
          },
          {
            "Action": [
              "vpc:*"
            ],
            "Resource": [
              "*"
            ],
            "Effect": "Allow"
          },
          {
            "Action": [
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
            "Action": [
              "ecs:AttachDisk",
              "ecs:DetachDisk",
              "ecs:DescribeDisks",
              "ecs:CreateDisk",
              "ecs:CreateSnapshot",
              "ecs:DeleteDisk",
              "ecs:CreateNetworkInterface",
              "ecs:DescribeNetworkInterfaces",
              "ecs:AttachNetworkInterface",
              "ecs:AssignPrivateIpAddresses",
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
            "Action": [
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
            "Action": [
              "eci:CreateContainerGroup",
              "eci:DeleteContainerGroup",
              "eci:DescribeContainerGroups",
              "eci:DescribeContainerLog"
            ],
            "Resource": ["*"],
            "Effect": "Allow"
          },
          { "Action": [ "log:*" ], "Resource": [ "*" ], "Effect": "Allow" },
          { "Action": [ "cms:*" ], "Resource": [ "*" ], "Effect": "Allow" },
          { "Action": [ "vpc:*" ], "Resource": [ "*" ], "Effect": "Allow" }
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
    "create_master_data_disk": {
      "Fn::Equals": [
        true,
        {
          "Ref": "MasterDataDisk"
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
    "master_no_keypair": {
      "Fn::Equals": [
        "",
        {
          "Ref": "MasterKeyPair"
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
              "Action": "sts:AssumeRole",
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
              "Action": "sts:AssumeRole",
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
      "Type": "ALIYUN::VPC::NatGateway",
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
        "AddressType": "internet",
        "LoadBalancerSpec": "slb.s1.small",
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
        "Bandwidth": -1
      }
    },
    "k8s_master_listener_boot": {
      "Type": "ALIYUN::SLB::Listener",
      "Properties": {
        "ListenerPort": 9443,
        "Bandwidth": -1,
        "LoadBalancerId": {
          "Ref": "k8s_master_slb"
        },
        "Protocol": "tcp",
        "HealthCheck": {
          "Interval": 10,
          "Port": 32443
        },
        "BackendServerPort": 32443
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
        "HealthCheck": {
          "Interval": 10,
          "Port": 6443
        },
        "BackendServerPort": 6443
      }
    },
    "k8s_master_sg": {
      "Type": "ALIYUN::ESS::ScalingGroup",
      "DependsOn": ["k8s_master_listener_boot","k8s_master_slb_listener","k8s_sg"],
      "Properties": {
        "MinSize": "1",
        "MaxSize": "20",
        "DefaultCooldown": 0,
        "ScalingGroupName": {
          "Fn::Join": [
            "-",
            [
              "kubernetes-master",
              {
                "Ref": "ALIYUN::StackId"
              }
            ]
          ]
        },
        "GroupDeletionProtection": false,
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
        "LoadBalancerIds": [
          { "Ref": "k8s_master_slb"},
          { "Fn::If": ["create_public_slb", {"Ref": "k8s_master_slb_internet"},{"Ref":"ALIYUN::NoValue"}]}
        ],
        "MultiAZPolicy": "BALANCE",
        "ProtectedInstances": {
          "Ref": "ProtectedInstances"
        },
        "RemovalPolicys": ["OldestScalingConfiguration", "NewestInstance"],
        "HealthCheckType": "ECS"
      }
    },
    "k8s_master_sconfig": {
      "Type": "ALIYUN::ESS::ScalingConfiguration",
      "Properties": {
        "ScalingGroupId": {
          "Ref": "k8s_master_sg"
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
        "SecurityGroupId": {
          "Ref": "k8s_sg"
        },
        "ImageId": {
          "Ref": "MasterImageId"
        },
        "InstanceTypes": [{
          "Ref": "MasterInstanceType"
        }],
        "SystemDiskSize": {
          "Ref": "MasterSystemDiskSize"
        },
        "SystemDiskCategory": {
          "Ref": "MasterSystemDiskCategory"
        },
        "InstanceName": {
          "Fn::Join": [
            "-",
            [
              "k8s-master",
              {
                "Ref": "ALIYUN::StackName"
              }
            ]
          ]
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
    "k8s_master_srule": {
      "Type": "ALIYUN::ESS::ScalingRule",
      "Properties": {
        "AdjustmentType": "TotalCapacity",
        "ScalingGroupId": { "Ref": "k8s_master_sg" },
        "AdjustmentValue": 1
      }
    },
    "k8s_master_sg_enable": {
      "Type": "ALIYUN::ESS::ScalingGroupEnable",
      "DependsOn": ["k8s_master_listener_boot","k8s_master_slb_listener","k8s_sg"],
      "Properties": {
        "ScalingGroupId": { "Ref": "k8s_master_sg" },
        "ScalingConfigurationId": { "Ref": "k8s_master_sconfig" },
        "ScalingRuleAris": [
          {
            "Fn::GetAtt": [
              "k8s_master_srule",
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
    },
    "k8s_master_waiter_handle": {
      "Type": "ALIYUN::ROS::WaitConditionHandle"
    },
    "k8s_master_waiter": {
      "Type": "ALIYUN::ROS::WaitCondition",
      "Properties": {
        "Timeout": 300,
        "Count": 1,
        "Handle": {
          "Ref": "k8s_master_waiter_handle"
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
          },
          {
            "Description" : "This is used by kubernetes. Do not delete please!",
            "SourceCidrIp": "100.104.0.0/16",
            "IpProtocol": "all",
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
    }
  },
  "Outputs": {
    "APIServerIntranet": {
      "Value": {
          "Fn::GetAtt": [
            "k8s_master_slb",
            "IpAddress"
          ]
      },
      "Description": "API Server Inner IP"
    },
    "APIServerIntranetIP": {
      "Value": {
        "Fn::GetAtt": [
          "k8s_master_slb",
          "IpAddress"
        ]
      },
      "Description": "API Server Inner IP"
    },
    "APIServerInternet": {
      "Description": "API Server Public IP",
      "Value": {
        "Fn::If": [
          "create_public_slb",
          {
            "Fn::GetAtt": ["k8s_master_slb_internet", "IpAddress"]
          },
          {
            "Ref":"ALIYUN::NoValue"
          }
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
      "Value": [],
      "Description": "Private IP Master node"
    },
    "MasterInstanceIDs": {
      "Value": [],
      "Description": "Ids of master node"
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
      "Value": {}
    }
  }
}
`
