package api

type OpenApiCommonResp struct {
	Code    interface{} `json:"Code"`
	Message interface{} `json:"Msg"`
	Data    interface{} `json:"Data,omitempty"`
	TaskId  interface{} `json:"TaskId,omitempty"`
}

type DescribeHaInstanceByTagReq struct {
	TagName string `json:"TagName"`
}

type DescribeInstanceDiskResponse struct {
	OpenApiCommonResp
	Data []DescribeHaInstanceData `json:"Data"`
}

type DescribeHaInstanceData struct {
	Cpu          int    `json:"Cpu"`
	Ram          int    `json:"Ram"`
	RegionId     string `json:"RegionId"`
	InstanceUuid string `json:"InstanceUuid"`
	VdcName      string `json:"VdcName"`
	Status       string `json:"Status"`
	CreatedTime  string `json:"CreatedTime"`
}

type BackendServer struct {
	IP      string `json:"IP"`
	Port    int    `json:"Port"`
	Weight  string `json:"Weight"`
	MaxConn int    `json:"MaxConn"`
}

type TcpListener struct {
	EnableSourceIP     string          `json:"EnableSourceIp"`
	EnableRepeaterMode string          `json:"EnableRepeaterMode"`
	ServerTimeoutUnit  string          `json:"ServerTimeoutUnit"`
	ACLWhiteList       []any           `json:"AclWhiteList"`
	ListenerMode       string          `json:"ListenerMode"`
	ListenerName       string          `json:"ListenerName"`
	Scheduler          string          `json:"Scheduler"`
	MaxConn            int             `json:"MaxConn"`
	ClientTimeoutUnit  string          `json:"ClientTimeoutUnit"`
	ListenerPort       int             `json:"ListenerPort"`
	ServerTimeout      string          `json:"ServerTimeout"`
	ConnectTimeoutUnit string          `json:"ConnectTimeoutUnit"`
	BackendServer      []BackendServer `json:"BackendServer"`
	ConnectTimeout     string          `json:"ConnectTimeout"`
	ClientTimeout      string          `json:"ClientTimeout"`
}

type HttpListener struct {
	ServerTimeoutUnit  string          `json:"ServerTimeoutUnit"`
	ServerTimeout      string          `json:"ServerTimeout"`
	StickySession      string          `json:"StickySession"`
	ACLWhiteList       []string        `json:"AclWhiteList"`
	Option             interface{}     `json:"Option"`
	SessionPersistence interface{}     `json:"SessionPersistence"`
	CertificateIds     interface{}     `json:"CertificateIds"`
	ListenerMode       string          `json:"ListenerMode"`
	MaxConn            int             `json:"MaxConn"`
	ConnectTimeoutUnit string          `json:"ConnectTimeoutUnit"`
	Scheduler          string          `json:"Scheduler"`
	BackendServer      []BackendServer `json:"BackendServer"`
	ConnectTimeout     string          `json:"ConnectTimeout"`
	ClientTimeout      string          `json:"ClientTimeout"`
	ListenerName       string          `json:"ListenerName"`
	ClientTimeoutUnit  string          `json:"ClientTimeoutUnit"`
	ListenerPort       int             `json:"ListenerPort"`
}

type HaStrategyInfoData struct {
	TCPListeners  []TcpListener  `json:"TcpListeners"`
	HttpListeners []HttpListener `json:"HttpListeners"`
}

type HaStrategyInfoDataResponse struct {
	OpenApiCommonResp
	Data HaStrategyInfoData `json:"Data"`
}

type ModifyHaStrategyReq struct {
	*HaStrategyInfoData
	InstanceUuid string `json:"InstanceUuid"`
}
