package api

import (
	"cds-k8s-tools/pkg/utils"
	"fmt"
	"net/http"
)

const Success = "Success"

// DescribeHaproxyInstancesByTag 获取Haproxy实例详情信息
func DescribeHaproxyInstancesByTag(params map[string]string) ([]DescribeHaInstanceData, error) {
	var resp = DescribeInstanceDiskResponse{}
	status, err := utils.NewOpenapiRequest(utils.DescribeHaInstance, http.MethodGet, params, "", &resp)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK || resp.Code != Success {
		return nil, fmt.Errorf("get haproxy instance failed, status: %v, code:%v, message: %v", status, resp.Code, resp.Message)
	}
	return resp.Data, nil
}

// DescribeLoadBalancerStrategy 获取Ha实例的当前监听的策略配置列表
func DescribeLoadBalancerStrategy(params map[string]string) (*HaStrategyInfoData, error) {
	var resp = HaStrategyInfoDataResponse{}
	status, err := utils.NewOpenapiRequest(utils.DescribeLoadBalancerStrategy, http.MethodGet, params, "", &resp)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK || resp.Code != Success {
		return nil, fmt.Errorf("get haproxy instance strategy failed, status: %v, code:%v, message: %v", status, resp.Code, resp.Message)
	}
	return &resp.Data, nil
}

// ModifyLoadBalancerStrategy 修改Ha实例的当前监听的策略配置列表，全量覆盖写
func ModifyLoadBalancerStrategy(params ModifyHaStrategyReq) (*OpenApiCommonResp, error) {
	var resp = OpenApiCommonResp{}
	status, err := utils.NewOpenapiRequest(utils.ModifyLoadBalancerStrategy, http.MethodPost, nil, params, &resp)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK || resp.Code != Success {
		return nil, fmt.Errorf("modify haproxy instance strategy failed, status: %v, code:%v, message: %v", status, resp.Code, resp.Message)
	}
	return &resp, nil
}
