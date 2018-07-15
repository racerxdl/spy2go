package main

import (
	"github.com/racerxdl/spy2go"
	"fmt"
	"log"
	"time"
)


func OnFloatIQ(data []complex64) {
	log.Println("Received Complex 64 Data! ", len(data))
}
func OnInt16IQ(data []spy2go.ComplexInt16) {
	log.Println("Received Complex 16 Data! ", len(data))
}
func OnUint8IQ(data []spy2go.ComplexUInt8) {
	log.Println("Received Complex 8 Data! ", len(data))
}
func OnDeviceSync() {
	log.Println("Got device sync!")
}
func OnFFT(data []uint8) {
	log.Println("Received FFT! ", len(data))
}

func main() {
	var spyserver = spy2go.MakeSpyserver("10.10.5.147", 5555)

	var cb = spy2go.CallbackBase{
		OnFloatIQ: OnFloatIQ,
		OnDeviceSync: OnDeviceSync,
		OnInt16IQ: OnInt16IQ,
		OnUInt8IQ: OnUint8IQ,
		OnFFT: OnFFT,
	}

	spyserver.SetCallback(&cb)

	spyserver.Connect()

	log.Println(fmt.Sprintf("Device: %s", spyserver.GetName()))
	var srs = spyserver.GetAvailableSampleRates()

	log.Println("Available SampleRates:")
	for i := 0; i < len(srs); i++ {
		log.Println(fmt.Sprintf("		%f msps", float32(srs[i]) / 1e6))
	}
	if spyserver.SetSampleRate(6000000) == spy2go.InvalidValue {
		log.Println("Error setting sample rate.")
	}
	if spyserver.SetCenterFrequency(106300000) == spy2go.InvalidValue {
		log.Println("Error setting center frequency.")
	}

	spyserver.SetStreamingMode(spy2go.StreamModeFFTIQ)

	log.Println("Starting")
	spyserver.Start()

	time.Sleep(time.Second*10)

	log.Print("Stopping")
	spyserver.Stop()

	spyserver.Disconnect()
}