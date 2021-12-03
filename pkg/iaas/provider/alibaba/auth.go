/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package alibaba

import (
	b64 "encoding/base64"
	"encoding/json"
	"flag"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/denverdino/aliyungo/metadata"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/klog/v2"
	"path/filepath"
	"time"

	"fmt"
	"github.com/go-cmd/cmd"
	"k8s.io/apimachinery/pkg/util/wait"
	"strings"
)

var KUBERNETES_NODE_LIFE_CYCLE = "ack.ovm"

// TOKEN_RESYNC_PERIOD default Token sync period
var TOKEN_RESYNC_PERIOD = 10 * time.Minute

// ClientAuth client manager for aliyun sdk
type ClientAuth struct {
	stop <-chan struct{}

	Meta IMetaData
	ECS  *ecs.Client
}

type ProviderConfig struct {
	Region       string `json:"region,omitempty" protobuf:"bytes,2,opt,name=region"`
	AccessKey    string `json:"accessKey,omitempty" protobuf:"bytes,2,opt,name=accessKey"`
	AccessSecret string `json:"accessSecret,omitempty" protobuf:"bytes,2,opt,name=accessSecret"`
	UID          string `json:"uid,omitempty" protobuf:"bytes,2,opt,name=uid"`
}

var CFG = &ProviderConfig{}

var providerconfig string

func init() {
	// TODO: Fix this to allow double vendoring this library but still register flags on behalf of users
	flag.StringVar(&providerconfig, "provider-config", "",
		"Paths to a kubeconfig. Only required if out-of-cluster.")
}

// NewClientMgr return a new client manager
func NewClientAuth() *ClientAuth {
	eclient, err := ecs.NewClientWithAccessKey("cn-hangzhou", "key", "secret")
	if err != nil {
		panic(errors.Wrap(err, "new client auth"))
	}
	return &ClientAuth{
		Meta: NewMetaData(),
		ECS:  eclient,
		stop: make(<-chan struct{}, 1),
	}
}

func (mgr *ClientAuth) Start(
	settoken func(mgr *ClientAuth, token *Token) error,
) error {
	initialized := false
	tokenfunc := func() {
		// reload config while token refresh
		err := LoadCfg(providerconfig)
		if err != nil {
			klog.Warningf("load config fail: %s", err.Error())
			return
		}
		// refresh client Token periodically
		token, err := mgr.Token().NextToken()
		if err != nil {
			klog.Errorf("return next token: %s", err.Error())
			return
		}
		err = settoken(mgr, token)
		if err != nil {
			klog.Errorf("set Token: %s", err.Error())
			return
		}
		initialized = true
	}
	go wait.Until(
		tokenfunc,
		TOKEN_RESYNC_PERIOD,
		mgr.stop,
	)
	return wait.ExponentialBackoff(
		wait.Backoff{
			Steps:    7,
			Duration: 1 * time.Second,
			Jitter:   1,
			Factor:   2,
		}, func() (done bool, err error) {
			tokenfunc()
			klog.Infof("wait for Token ready")
			return initialized, nil
		},
	)
}

func LoadCfg(cfg string) error {
	content, err := ioutil.ReadFile(cfg)
	if err != nil {
		return fmt.Errorf("read config file: %s", content)
	}
	return yaml.Unmarshal(content, CFG)
}

func (mgr *ClientAuth) Token() TokenAuth {
	key, err := b64.StdEncoding.DecodeString(CFG.AccessKey)
	if err != nil {
		panic(fmt.Sprintf("ak must be base64 encoded: %s", err.Error()))
	}
	secret, err := b64.StdEncoding.DecodeString(CFG.AccessSecret)
	if err != nil {
		panic(fmt.Sprintf("ak must be base64 encoded: %s", err.Error()))
	}
	if len(key) == 0 ||
		len(secret) == 0 {
		klog.Infof("nlc: use ramrole Token mode without ak.")
		return &RamRoleToken{meta: mgr.Meta}
	}
	region := CFG.Region
	if region == "" {
		region, err = mgr.Meta.Region()
		if err != nil {
			panic(fmt.Sprintf("region not specified in config, detect region failed: %s", err.Error()))
		}
	}
	inittoken := &Token{
		AccessKey:    string(key),
		AccessSecret: string(secret),
		UID:          CFG.UID,
		Region:       region,
	}
	if inittoken.UID == "" {
		klog.Infof("nlc: ak mode to authenticate user. without Token and role assume")
		return &AkAuthToken{ak: inittoken}
	}
	klog.Infof("nlc: service account auth mode")
	return &ServiceToken{svcak: inittoken}
}

func RefreshToken(mgr *ClientAuth, token *Token) error {
	//mgr.ECS.WithSecurityToken(token.Token).
	//	WithAccessKeyId(token.AccessKey).
	//	WithRegionID(common.Region(token.Region)).
	//	WithAccessKeySecret(token.AccessSecret)
	//
	//mgr.ECS.SetUserAgent(KUBERNETES_NODE_LIFE_CYCLE)
	return nil
}

// MetaData return MetaData client
func (mgr *ClientAuth) MetaData() IMetaData { return mgr.Meta }

// Token base Token info
type Token struct {
	Region       string `json:"region,omitempty"`
	AccessSecret string `json:"accessSecret,omitempty"`
	UID          string `json:"uid,omitempty"`
	Token        string `json:"token,omitempty"`
	AccessKey    string `json:"accesskey,omitempty"`
}

// TokenAuth is an interface of Token auth method
type TokenAuth interface {
	NextToken() (*Token, error)
}

// AkAuthToken implement ak auth
type AkAuthToken struct{ ak *Token }

func (f *AkAuthToken) NextToken() (*Token, error) { return f.ak, nil }

type RamRoleToken struct {
	meta IMetaData
}

func (f *RamRoleToken) NextToken() (*Token, error) {
	roleName, err := f.meta.RoleName()
	if err != nil {
		return nil, fmt.Errorf("role name: %s", err.Error())
	}
	// use instance ram file way.
	role, err := f.meta.RamRoleToken(roleName)
	if err != nil {
		return nil, fmt.Errorf("ramrole Token retrieve: %s", err.Error())
	}
	return &Token{
		AccessKey:    role.AccessKeyId,
		AccessSecret: role.AccessKeySecret,
		Token:        role.SecurityToken,
	}, nil
}

// ServiceToken is an implemention of service account auth
type ServiceToken struct {
	svcak    *Token
	execpath string
}

func (f *ServiceToken) NextToken() (*Token, error) {
	status := <-cmd.NewCmd(
		filepath.Join(f.execpath, "servicetoken"),
		fmt.Sprintf("--uid=%s", f.svcak.UID),
		fmt.Sprintf("--key=%s", f.svcak.AccessKey),
		fmt.Sprintf("--secret=%s", f.svcak.AccessSecret),
	).Start()
	if status.Error != nil {
		return nil, fmt.Errorf("invoke servicetoken: %s", status.Error.Error())
	}
	token := &Token{}
	err := json.Unmarshal(
		[]byte(strings.Join(status.Stdout, "")), token,
	)
	if err == nil {
		return token, nil
	}
	return nil, fmt.Errorf("unmarshal Token: %s, %s, %s", err.Error(), status.Stdout, status.Stderr)
}

// IMetaData metadata interface
type IMetaData interface {
	HostName() (string, error)
	ImageID() (string, error)
	InstanceID() (string, error)
	Mac() (string, error)
	NetworkType() (string, error)
	OwnerAccountID() (string, error)
	PrivateIPv4() (string, error)
	Region() (string, error)
	SerialNumber() (string, error)
	SourceAddress() (string, error)
	VpcCIDRBlock() (string, error)
	VpcID() (string, error)
	VswitchCIDRBlock() (string, error)
	Zone() (string, error)
	NTPConfigServers() ([]string, error)
	RoleName() (string, error)
	RamRoleToken(role string) (metadata.RoleAuth, error)
	VswitchID() (string, error)
}

// NewMetaData return new metadata
func NewMetaData() IMetaData {
	if false {
		// use mocked Meta depend
		klog.Infof("use mocked metadata server.")
		return &fakeMetaData{base: metadata.NewMetaData(nil)}
	}
	return metadata.NewMetaData(nil)
}

type fakeMetaData struct {
	base IMetaData
}

func (m *fakeMetaData) HostName() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) ImageID() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) InstanceID() (string, error) {

	return "fakedInstanceid", nil
}

func (m *fakeMetaData) Mac() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) NetworkType() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) OwnerAccountID() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) PrivateIPv4() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) Region() (string, error) {
	return m.base.Region()
}

func (m *fakeMetaData) SerialNumber() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) SourceAddress() (string, error) {

	return "", fmt.Errorf("unimplemented")

}

func (m *fakeMetaData) VpcCIDRBlock() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) VpcID() (string, error) {
	return m.base.VpcID()
}

func (m *fakeMetaData) VswitchCIDRBlock() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

// zone1:vswitchid1,zone2:vswitch2
func (m *fakeMetaData) VswitchID() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) EIPv4() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) DNSNameServers() ([]string, error) {

	return []string{""}, fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) NTPConfigServers() ([]string, error) {

	return []string{""}, fmt.Errorf("unimplemented")
}

func (m *fakeMetaData) Zone() (string, error) {
	return m.base.Zone()
}

func (m *fakeMetaData) RoleName() (string, error) {

	return m.base.RoleName()
}

func (m *fakeMetaData) RamRoleToken(role string) (metadata.RoleAuth, error) {

	return m.base.RamRoleToken(role)
}
