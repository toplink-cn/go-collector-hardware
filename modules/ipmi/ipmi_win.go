//go:build windows
// +build windows

package ipmi

import (
	"fmt"

	"github.com/yusufpapurcu/wmi"
)

type IPMI_SensorData struct {
	Name             string
	SensorType       string
	CurrentReading   float64
	Units            string
	LowerThreshold   float64
	UpperThreshold   float64
	Enabled          bool
	TimeOfLastRead   string
	OperationalState uint16
}

func GetInfo() []*Sensor {

	sensors := []*Sensor{}

	var sensorData []IPMI_SensorData
	err := wmi.QueryNamespace("SELECT * FROM MSFT_Sensor", &sensorData, "ROOT\\WMI")
	if err != nil {
		panic(err)
	}

	fmt.Println("len:", len(sensorData))
	if len(sensorData) < 1 {
		return sensors
	}

	for _, item := range sensorData {
		fmt.Printf("Name: %s\n", item.Name)
		fmt.Printf("Type: %s\n", item.SensorType)
		fmt.Printf("Reading: %f %s\n", item.CurrentReading, item.Units)
		fmt.Printf("Lower Threshold: %f\n", item.LowerThreshold)
		fmt.Printf("Upper Threshold: %f\n", item.UpperThreshold)
		fmt.Printf("Enabled: %t\n", item.Enabled)
		fmt.Printf("Time of Last Read: %s\n", item.TimeOfLastRead)
		fmt.Printf("Operational State: %d\n\n", item.OperationalState)
	}

	return sensors
}
