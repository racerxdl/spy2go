package main

import (
	"github.com/racerxdl/spy2go/airspy"
	"github.com/racerxdl/spy2go/spytypes"
	"log"
	"time"
)

type MyCallback struct {}

func (cb *MyCallback) OnData(dType int, data interface{}) {
	if dType == spytypes.SamplesComplex64 {
		samples := data.([]complex64)
		log.Println("Received Complex 64 bit Data! ", len(samples))
	} else if dType == spytypes.SamplesComplex32 {
		samples := data.([]spytypes.ComplexInt16)
		log.Println("Received Complex 32 bit Data! ", len(samples))
	} else if dType == spytypes.SamplesComplexUInt8 {
		samples := data.([]spytypes.ComplexUInt8)
		log.Println("Received Complex 64 Data! ", len(samples))
	} else if dType == spytypes.SamplesBytes {
		samples := data.([]byte)
		log.Println("Received Raw Data! ", len(samples))
	}
}


func main() {
	cb := MyCallback{}
	airspy.Initialize()
	log.Println(airspy.GetLibraryVersion())

	dev := airspy.MakeAirspyDevice(0)
	dev.SetCallback(&cb)

	log.Printf("Got %s\n", dev.GetName())

	dev.Start()

	time.Sleep(time.Second * 2)

	dev.Stop()

	dev.Close()

	airspy.DeInitialize()
}