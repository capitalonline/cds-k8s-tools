package service

type CommonResp struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

var (
	Success           = CommonResp{"Success", "成功", nil}
	ParamError        = CommonResp{"ParamError", "参数错误", nil}
	AlarmServiceError = CommonResp{"AlarmServiceError", "告警服务异常", nil}
	AlarmServiceWarn  = CommonResp{"AlarmServiceWarn", "告警服务错误", nil}
)

func ReturnCommonResp(base CommonResp, data interface{}) CommonResp {
	return CommonResp{
		Code: base.Code,
		Msg:  base.Msg,
		Data: data,
	}
}
