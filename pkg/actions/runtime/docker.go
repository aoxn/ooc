//go:build linux || darwin
// +build linux darwin

package runtime

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/actions"
	"github.com/aoxn/wdrip/pkg/utils/cmd"
	"io/ioutil"
	"k8s.io/klog/v2"
	"os"
)

type action struct{}

// NewAction returns a new action for kubeadm init
func NewAction() actions.Action {
	return &action{}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	err := os.MkdirAll("/etc/docker", 0755)
	if err != nil {
		return fmt.Errorf("make /etc/docker dir: %s", err.Error())
	}
	err = os.MkdirAll("/etc/containerd", 0755)
	if err != nil {
		return fmt.Errorf("make /etc/docker dir: %s", err.Error())
	}
	for f, v := range cfgs {
		err = ioutil.WriteFile(f, []byte(v), 0755)
		if err != nil {
			return fmt.Errorf("write docker config: %s", err.Error())
		}
	}
	klog.Infof("add docker group...")
	sta := <-cmd.NewCmd("groupadd", "-r", "docker").Start()
	if err := cmd.CmdError(sta); err != nil {
		klog.Errorf("add docker group error: %s", err.Error())
	}
	err = cmd.Systemctl([]string{"enable", "docker"})
	if err != nil {
		return fmt.Errorf("systecmctl enable docker error,%s ", err.Error())
	}
	return cmd.Systemctl([]string{"start", "docker"})
}

var daemonjson = `
{
    "exec-opts": ["native.cgroupdriver=systemd"],
    "log-driver": "json-file",
    "log-opts": {
        "max-size": "100m",
        "max-file": "10"
    },
    "bip": "169.254.123.1/24",
    "oom-score-adjust": -1000,
    "registry-mirrors": [],
    "storage-driver": "overlay2",
    "storage-opts":["overlay2.override_kernel_check=true"],
    "live-restore": true
}
`

var nvidiadaemonjson = `
{
    "default-runtime": "nvidia",
    "runtimes": {
        "nvidia": {
            "path": "/usr/bin/nvidia-container-runtime",
            "runtimeArgs": []
        }
    },
    "exec-opts": ["native.cgroupdriver=systemd"],
    "log-driver": "json-file",
    "log-opts": {
        "max-size": "100m",
        "max-file": "10"
    },
    "bip": "169.254.123.1/24",
    "oom-score-adjust": -1000,
    "registry-mirrors": [""],
    "storage-driver": "overlay2",
    "storage-opts":["overlay2.override_kernel_check=true"],
    "live-restore": true
}
`

type dockerDaemonJson struct {
	runtime string

	execOpts        []string
	logDriver       string
	logOpts         logOpt
	bip             string
	oomScore        int
	registryMirrors []string
	storageDriver   string
	storageOpts     []string
	liveRestore     bool
}

type logOpt struct {
	maxSize string
	maxFile string
}

var cfgs = map[string]string{
	"/lib/systemd/system/docker.service":     dockerunit,
	"/lib/systemd/system/docker.socket":      dockersock,
	"/etc/containerd/config.toml":            containerdcfg,
	"/lib/systemd/system/containerd.service": containerdsvc,
	"/etc/docker/daemon.json":                daemonjson,
}

var dockerunit = `
[Unit]
Description=Docker Application Container Engine
Documentation=https://docs.docker.com
BindsTo=containerd.service
After=network-online.target firewalld.service containerd.service
Wants=network-online.target
Requires=docker.socket

[Service]
Type=notify
ExecStart=/usr/bin/dockerd -H fd:// --containerd=/run/containerd/containerd.sock
ExecStartPost=/usr/sbin/iptables -P FORWARD ACCEPT
ExecReload=/bin/kill -s HUP \$MAINPID
TimeoutSec=0
RestartSec=2
Restart=always
StartLimitBurst=3
StartLimitInterval=60s
LimitNOFILE=infinity
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
Delegate=yes
KillMode=process

[Install]
WantedBy=multi-user.target
`

var dockersock = `
[Unit]
Description=Docker Socket for the API
PartOf=docker.service

[Socket]
ListenStream=/var/run/docker.sock
SocketMode=0660
SocketUser=root
SocketGroup=docker

[Install]
WantedBy=sockets.target
`

var containerdcfg = `
disabled_plugins = ["cri"]
oom_score = -999
`

var containerdsvc = `
[Unit]
Description=containerd container runtime
Documentation=https://containerd.io
After=network.target

[Service]
ExecStartPre=-/sbin/modprobe overlay
ExecStart=/usr/bin/containerd
KillMode=process
Delegate=yes
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity

[Install]
WantedBy=multi-user.target
`
