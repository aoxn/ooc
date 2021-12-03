package fake

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/aoxn/ooc/pkg/node/meta"
	alibaba "github.com/aoxn/ooc/pkg/node/meta/alibaba"
)

type Config struct {
	ZoneID    string
	Region    string
	VpcID     string
	VswitchID string
}

// NewMetaData return new metadata
func NewMetaData(cfg *Config) meta.Meta {
	if cfg.VpcID != "" &&
		cfg.VswitchID != "" {
		klog.Infof("use mocked metadata server.")
		return &fakeMetaData{
			config: cfg,
			base:   alibaba.NewMetaDataAlibaba(nil),
		}
	}
	return alibaba.NewMetaDataAlibaba(nil)
}

type fakeMetaData struct {
	config *Config
	base   meta.Meta
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
	if m.config.Region != "" {
		return m.config.Region, nil
	}
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
	if m.config.VpcID != "" {
		return m.config.VpcID, nil
	}
	return m.base.VpcID()
}

func (m *fakeMetaData) VswitchCIDRBlock() (string, error) {

	return "", fmt.Errorf("unimplemented")
}

// zone1:vswitchid1,zone2:vswitch2
func (m *fakeMetaData) VswitchID() (string, error) {

	if m.config.VswitchID == "" {
		// get vswitch id from meta server
		return m.base.VswitchID()
	}
	zlist := strings.Split(m.config.VswitchID, ",")
	if len(zlist) == 1 {
		klog.Infof("simple vswitchid mode, %s", m.config.VswitchID)
		return m.config.VswitchID, nil
	}
	zone, err := m.Zone()
	if err != nil {
		return "", fmt.Errorf("retrieve vswitchid error for %s", err.Error())
	}
	for _, zone := range zlist {
		vs := strings.Split(zone, ":")
		if len(vs) != 2 {
			return "", fmt.Errorf("cloud-config vswitch format error: %s", m.config.VswitchID)
		}
		if vs[0] == zone {
			return vs[1], nil
		}
	}
	klog.Infof("zone[%s] match failed, fallback with simple vswitch id mode, [%s]", zone, m.config.VswitchID)
	return m.config.VswitchID, nil
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
	if m.config.ZoneID != "" {
		return m.config.ZoneID, nil
	}
	return m.base.Zone()
}

func (m *fakeMetaData) RoleName() (string, error) {

	return m.base.RoleName()
}

func (m *fakeMetaData) RamRoleToken(role string) (meta.RoleAuth, error) {

	return m.base.RamRoleToken(role)
}
