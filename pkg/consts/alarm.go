package consts

const (
	SendAlarm         = "SendAlarm"
	SendSNatAlarmInfo = "SendSnatAlarmInfo"
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
)

const (
	SNatErrorAlarmType   = "snat.error"
	SNatRecoverAlarmType = "snat.recover"
)
