

```bash
wdrip vm -a "console=hvc0" \
       -k /Users/aoxn/.wdrip/.vm/vm-ubuntu01/vmlinux \
       -i /Users/aoxn/.wdrip/.vm/vm-ubuntu01/initrd \
       -d /Users/aoxn/.wdrip/.vm/vm-ubuntu01/focal-server-cloudimg-amd64.img \
       -d /Users/aoxn/.wdrip/.vm/vm-ubuntu01/data.img \
       -p unix@/tmp/aoxn.sock:2735
```

```bash
touch /etc/cloud/cloud-init.disabled
echo 'root:root' | chpasswd
echo "podman" >/etc/hostname
ssh-keygen -f /etc/ssh/ssh_host_rsa_key -N '' -t rsa
ssh-keygen -f /etc/ssh/ssh_host_dsa_key -N '' -t dsa
ssh-keygen -f /etc/ssh/ssh_host_ed25519_key -N '' -t ed25519
cat <<EOF > /etc/netplan/01-dhcp.yaml 
network:
    renderer: networkd
    ethernets:
        enp0s1:
            dhcp4: no
            addresses: [192.168.64.2/24]
            gateway4: 192.168.64.1
            nameservers:
                addresses: [114.114.114.114]
    version: 2
EOF
echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
sed -i "/^PasswordAuthentication/ c PasswordAuthentication yes" /etc/ssh/sshd_config

# disable
systemctl disable --now snapd.service snapd.socket
resize2fs /dev/vda
apt remove -y cloud-init cloud-initramfs-copymods cloud-initramfs-dyn-netconf cloud-guest-utils popularity-contest


cat <<EOF > /etc/apt/sources.list
deb http://mirrors.aliyun.com/ubuntu/ focal main restricted
deb http://mirrors.aliyun.com/ubuntu/ focal-updates main restricted
deb http://mirrors.aliyun.com/ubuntu/ focal universe
deb http://mirrors.aliyun.com/ubuntu/ focal-updates universe
deb http://mirrors.aliyun.com/ubuntu/ focal multiverse
deb http://mirrors.aliyun.com/ubuntu/ focal-updates multiverse
deb http://mirrors.aliyun.com/ubuntu/ focal-backports main restricted universe multiverse
deb http://mirrors.aliyun.com/ubuntu/ focal-security main restricted
deb http://mirrors.aliyun.com/ubuntu/ focal-security universe
deb http://mirrors.aliyun.com/ubuntu/ focal-security multiverse
EOF
```

sudo apt-get remove docker docker-engine docker.io containerd runc
sudo apt-get update
sudo apt-get install \
        ca-certificates \
        curl \
        gnupg \
        lsb-release
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io


./wdrip.amd64 vm proxy -p vsock@2375:unix@/var/run/docker.sock
