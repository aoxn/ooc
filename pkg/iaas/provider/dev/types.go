package dev

type ConfigurationROS struct {
	VPC VPC

	//LoadBalancer LoadBalancer

	Vswitchs []Vswitch
}

type VPC struct {
	Vpcid string
}

type Vswitch struct {
	VswitchID string
}
