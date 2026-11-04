package consts

const (
	SendAlarm           = "SendAlarm"
	SendSNatAlarmInfoV2 = "SendSnatAlarmInfoV2"
	SendPodStatusAlarm  = "SendPodStatusAlarm"
)

const (
	SuccessCode = "Success"
)

const (
	AlarmPodName               = "cds-alarm-service"
	AlarmPodNamespace          = "kube-system"
	AlarmPodServiceDefaultPort = 39989
	AlarmServiceV2Route        = "/v2/alarm"

	SNatPodName = "cds-snat-configuration"

	PodAlarmPodName = "cds-pod-alarm"
)

const (
	SNatErrorAlarmType   = "snat.error"
	SNatRecoverAlarmType = "snat.recover"
)
