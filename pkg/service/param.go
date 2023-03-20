package service

type AlarmReq struct {
	Name      string        `json:"name"`
	SubObject string        `json:"subObject"`
	Type      string        `json:"type"`
	Group     string        `json:"group"`
	Customer  string        `json:"customer"`
	Level     string        `json:"level"`
	Message   *AlarmMessage `json:"message"`
	Timestamp string        `json:"timestamp"`
	Ip        string        `json:"ip"`
}

type AlarmMessage struct {
	ClusterId string `json:"cluster_id"`
	NodeName  string `json:"node_name"`
	Status    string `json:"status"` // error  recover
	Msg       string `json:"msg"`
}

type CckAlarmReq struct {
	ClusterId  string `json:"cluster_id"`
	Site       string `json:"site"`
	Msg        string `json:"msg"`
	Hostname   string `json:"hostname,omitempty"`
	NtfName    string `json:"ntf_name,omitempty"`
	AlarmType  string `json:"alarm_type,omitempty"`
	AlarmGroup string `json:"alarm_group,omitempty"`
	Ip         string `json:"ip"`
	Tag1       string `json:"tag1"`
}
