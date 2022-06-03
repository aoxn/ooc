package e2e

import (
	"encoding/json"
	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ess"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"testing"
	"time"
)

func init() {
	ViperizeFlags()
}

func TestE2E(t *testing.T) {
	time1 := "2019-08-18T08:42:29"

	time2 := "2019-08-18T08:50:29"

	evtOut, err := time.Parse("2006-01-02T15:04:05", time1)
	if err != nil {
		panic(err)
	}
	evtIn, err := time.Parse("2006-01-02T15:04:05", time2)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%t", evtOut.After(evtIn))

	data, err := ioutil.ReadFile("/Users/aoxn/work/wdrip/pkg/iaas/provider/ros/demo.alibaba.json")
	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}

	fin, err := yaml.JSONToYAML(data)
	if err != nil {
		t.Fatalf("yaml: %s", err.Error())
	}

	//data, err = yaml.YAMLToJSON(fin)
	fmt.Printf("FINAL: %s\n", fin)

}

func TestESS(t *testing.T) {

	count := -2
	// cn-shanghai asg
	//region := common.Shanghai
	//ruleid := "asr-uf68fsg4qpd3kvqeq57s"

	//cn-beijing asg
	region := common.Beijing
	ruleid := "asr-2ze4e7lqyl669dk2vxrh"

	client := ess.NewClient("", "")
	fmt.Printf("start at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	rules, _, err := client.DescribeScalingRules(
		&ess.DescribeScalingRulesArgs{
			RegionId:      region,
			ScalingRuleId: common.FlattenArray([]string{ruleid}),
		},
	)
	if err != nil {
		t.Fatalf("find scaling rule detail: %s", err.Error())
	}
	if len(rules) != 1 {
		t.Fatalf("multiple scaling rule found: %d by id %s", len(rules), ruleid)
	}

	_, err = client.ModifyScalingRule(
		&ess.ModifyScalingRuleArgs{
			RegionId:        region,
			ScalingRuleId:   ruleid,
			AdjustmentType:  ess.QuantityChangeInCapacity,
			AdjustmentValue: count,
		},
	)
	if err != nil {
		t.Fatalf("set scaling rule to %d fail: %s", count, err.Error())
	}
	_, err = client.ExecuteScalingRule(
		&ess.ExecuteScalingRuleArgs{
			ScalingRuleAri: rules[0].ScalingRuleAri,
		},
	)
	if err != nil {
		t.Fatalf("set scaling rule to %d fail: %s", count, err.Error())
	}
}

func TestNextRetry(t *testing.T) {
	next := NextRetry(1 * time.Second)
	for i := 0; i < 10; i++ {
		c := next()
		fmt.Printf("A=%d\n", c)
	}
}

func NextRetry(n time.Duration) func() time.Duration {
	start := n
	return func() time.Duration {
		start = 2 * start
		if start > 2*time.Minute {
			start = n
		}
		return start
	}
}

func TestModi(t *testing.T) {
	abc := `myselft
`
	contj, err := yaml.YAMLToJSON([]byte(abc))
	fmt.Println(string(contj))
	m := make(map[string]interface{})
	err = json.Unmarshal(contj, &m)

	if err != nil {
		t.Fatalf("unstructed marshal: %s", err.Error())
	}
	fmt.Println(m)
}
