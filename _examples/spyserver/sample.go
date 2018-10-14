package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/racerxdl/spy2go/spyserver"
	"github.com/racerxdl/spy2go/spytypes"
	"log"
	"os"
	"time"
)

var f *os.File

type MyCallback struct {}

func (cb *MyCallback) OnData(dType int, data interface{}) {
	if dType == spytypes.SamplesComplex64 {
		samples := data.([]complex64)
		log.Println("Received Complex 64 bit Data! ", len(samples))
	} else if dType == spytypes.SamplesComplex32 {
		samples := data.([]spytypes.ComplexInt16)
		log.Println("Received Complex 32 bit Data! ", len(samples))
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, data)

		f.Write(buf.Bytes())
	} else if dType == spytypes.SamplesComplexUInt8 {
		samples := data.([]spytypes.ComplexUInt8)
		log.Println("Received Complex 64 Data! ", len(samples))
	} else if dType == spytypes.FFTUInt8 {
		samples := data.([]complex64)
		log.Println("Received FFT Data! ", len(samples))
	} else if dType == spytypes.DeviceSync {
		log.Println("Got device sync!")
	}
}

func main() {
	var ss = spyserver.MakeSpyserver("10.10.5.147", 5555)

	var cb = MyCallback{}

	if f == nil {
		f, _ = os.Create("iq.raw")
	}

	ss.SetCallback(&cb)

	ss.Connect()

	log.Println(fmt.Sprintf("Device: %s", ss.GetName()))
	var srs = ss.GetAvailableSampleRates()

	log.Println("Available SampleRates:")
	for i := 0; i < len(srs); i++ {
		log.Println(fmt.Sprintf("		%f msps", float32(srs[i]) / 1e6))
	}
	if ss.SetSampleRate(750000) == spyserver.InvalidValue {
		log.Println("Error setting sample rate.")
	}
	if ss.SetCenterFrequency(106300000) == spyserver.InvalidValue {
		log.Println("Error setting center frequency.")
	}

	ss.SetStreamingMode(spyserver.StreamModeIQOnly)

	log.Println("Starting")
	ss.Start()

	time.Sleep(time.Second*10)

	log.Print("Stopping")
	ss.Stop()

	ss.Disconnect()
	f.Sync()
}