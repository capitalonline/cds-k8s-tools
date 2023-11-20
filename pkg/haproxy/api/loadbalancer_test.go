package api

import (
	"fmt"
	"github.com/gogf/gf/v2/util/gutil"
	"testing"
)

func TestDescribeHaInfo(t *testing.T) {
	params := map[string]string{
		"TagName": "reload_lb",
	}
	out, err := DescribeHaproxyInstancesByTag(params)
	if err != nil {
		fmt.Println(err)
	} else {
		gutil.Dump(out)
	}
}

func TestDescribeLoadBalancerStrategy(t *testing.T) {
	params := map[string]string{
		"InstanceUuid": "0ab23f0a-8520-11ee-bcd7-c2d101824e3b",
	}
	out, err := DescribeLoadBalancerStrategy(params)
	if err != nil {
		fmt.Println(err)
	} else {
		gutil.Dump(out)
	}
}

func TestModifyLoadBalancerStrategy(t *testing.T) {
	params := map[string]string{
		"InstanceUuid": "ea772da6-8293-11ee-bd32-72919a47465e",
	}
	Listeners, err := DescribeLoadBalancerStrategy(params)
	if err != nil {
		fmt.Println(err)
	}

	var newTcpListeners []TcpListener
	for _, policy := range Listeners.TCPListeners {
		if policy.ListenerName == "ccp-cluster-master-api" {
			policy.BackendServer = append(policy.BackendServer, BackendServer{
				IP:      "10.240.40.9",
				Port:    6443,
				Weight:  "1",
				MaxConn: 30000,
			})
		}
		newTcpListeners = append(newTcpListeners, policy)
	}
	Listeners.TCPListeners = newTcpListeners

	fmt.Println("begin modify haproxy policy")
	modifyParams := ModifyHaStrategyReq{
		InstanceUuid:       "ea772da6-8293-11ee-bd32-72919a47465e",
		HaStrategyInfoData: Listeners,
	}
	out, err := ModifyLoadBalancerStrategy(modifyParams)
	if err != nil {
		fmt.Println(err)
	} else {
		gutil.Dump(out)
	}
}
