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
	Type      string `json:"status"`
	Metric    string `json:"key"`
	Target    string `json:"target"`
	Value     string `json:"value"`
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

type CckNewAlarmReq struct {
	ClusterId      string          `json:"cluster_id"`
	AlarmInstances []AlarmInstance `json:"alarm_instances"`
}

type AlarmInstance struct {
	Type        string `json:"type"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Target      string `json:"target"`
	Instance    string `json:"instance"`
	Description string `json:"description"`
	EventTime   string `json:"event_time"`
}
