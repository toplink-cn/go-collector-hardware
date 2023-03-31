package cpu

type CpuObj struct {
	Usage       []CpuAtrr
	Temperature []CpuAtrr
}

type CpuAtrr struct {
	ID    string
	Value string
}
