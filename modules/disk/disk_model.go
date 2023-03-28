package disk

type Smartctl struct {
	Devices []Device `json:"devices"`
}

type Device struct {
	Name     string `json:"name"`
	InfoName string `json:"info_name"`
	Type     string `json:"type"`
	Protocol string `json:"protocol"`
}

type DiskInfo struct {
	ModelName    string       `json:"model_name"`
	SmartStatus  SmartStatus  `json:"smart_status"`
	UserCapacity UserCapacity `json:"user_capacity"`
	Temperature  Temperature  `json:"temperature"`
	PowerOnTime  PowerOnTime  `json:"power_on_time"`
}

type SmartStatus struct {
	Passed bool `json:"passed"`
}

type UserCapacity struct {
	Blocks int64 `json:"blocks"`
	Bytes  int64 `json:"bytes"`
}

type Temperature struct {
	Current int8 `json:"current"`
}

type PowerOnTime struct {
	Hours int64 `json:"hours"`
}
