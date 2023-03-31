package cpu

type CpuObj struct {
	Usage       []CpuAttr
	Temperature []CpuAttr
}

type CpuAttr struct {
	ID    string
	Value string
}
