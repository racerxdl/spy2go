package spy2go

import (
	"log"
	"net"
	"fmt"
	"time"
	"encoding/binary"
	"bytes"
)

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

type Spyserver struct {

	hostname string
	port int
	callback *CallbackBase
	client net.Conn

	terminated bool
	routineRunning bool
	gotDeviceInfo bool
	gotSyncInfo bool
	streamingMode uint32
	gain uint32

	availableSampleRates []uint32

	parserPhase uint32
	deviceInfo DeviceInfo
	header MessageHeader
	lastSequenceNumber uint32
	droppedBuffers uint32
	downStreamBytes uint64
	parserPosition uint32
	bodyBuffer []uint8
	headerBuffer []uint8

	Streaming bool
	CanControl bool
	IsConnected bool
	DroppedBuffers uint32
}

func MakeSpyserver(hostname string, port int) *Spyserver {
	return &Spyserver{
		hostname: hostname,
		port: port,
		callback: nil,
		terminated: false,
		gotDeviceInfo: false,
		gotSyncInfo: false,
		parserPhase: ParserAcquiringHeader,
		Streaming: false,
		CanControl: false,
		IsConnected: false,
		availableSampleRates: []uint32{},
		headerBuffer: make([]uint8, MessageHeaderSize),
	}
}

// region Private Methods
func (f *Spyserver) sayHello() bool {
	var totalLength = 4
	var softwareVersionBytes = []byte(SoftwareID)
	totalLength += len(softwareVersionBytes)


	var argBytes = make([]uint8, totalLength)
	binary.LittleEndian.PutUint32(argBytes[0:], SpyserverProtocolVersion)
	for i := 4; i < totalLength; i++ {
		argBytes[i] = softwareVersionBytes[i-4]
	}

	return f.sendCommand(CmdHello, argBytes)
}
func (f *Spyserver) cleanup() {
	f.deviceInfo.DeviceType = DeviceInvalid
	f.deviceInfo.DeviceSerial = 0
	f.deviceInfo.DecimationStageCount = 0
	f.deviceInfo.GainStageCount = 0
	f.deviceInfo.MaximumSampleRate = 0
	f.deviceInfo.MaximumBandwidth = 0
	f.deviceInfo.MaximumGainIndex = 0
	f.deviceInfo.MinimumFrequency = 0
	f.deviceInfo.MaximumFrequency = 0

	f.gain = 0
	f.CanControl = false
	f.gotDeviceInfo = false
	f.gotSyncInfo = false

	f.lastSequenceNumber = 0xFFFFFFFF
	f.droppedBuffers = 0
	f.downStreamBytes = 0
	f.parserPhase = ParserAcquiringHeader
	f.parserPosition = 0

	f.Streaming = false
	f.terminated = true
}
func (f *Spyserver) onConnect() {
	f.setSetting(SettingStreamingMode, []uint32 { f.streamingMode })
	f.setSetting(SettingIqFormat, []uint32 { StreamFormatInt16 })
	f.setSetting(SettingFFTFormat, []uint32 { StreamFormatUint8 })

	var sampleRates = make([]uint32, f.deviceInfo.DecimationStageCount)
	for i := uint32(0); i < f.deviceInfo.DecimationStageCount; i++ {
		sampleRates[i] = uint32(float32(f.deviceInfo.MaximumSampleRate) / float32(1 << i))
	}
}
func (f *Spyserver) setSetting(settingType uint32, params []uint32) bool {
	var argBytes = make([]uint8, 0)

	if len(params) > 0 {
		argBytes = make([]uint8, 4 + len(params) * 4)
		binary.LittleEndian.PutUint32(argBytes[0:], settingType)
		for i := 0; i < len(params); i++ {
			binary.LittleEndian.PutUint32(argBytes[i*4:], params[i])
		}
	}

	return f.sendCommand(CmdSetSetting, argBytes)
}
func (f *Spyserver) sendCommand(cmd uint32, args []uint8) bool {
	return false
}
func (f *Spyserver) parseMessage(buffer []uint8) {
	f.downStreamBytes++

	consumed := 0
	for len(buffer) > 0 && !f.terminated {
		if f.parserPhase == ParserAcquiringHeader {
			for f.parserPhase == ParserAcquiringHeader && len(buffer) > 0 {
				consumed = f.parseHeader(buffer)
				buffer = buffer[consumed:]
			}

			if f.parserPhase == ParserReadingData {
				clientMajor := uint8((SpyserverProtocolVersion >> 24) & 0xFF)
				clientMinor := uint8((SpyserverProtocolVersion >> 16) & 0xFF)

				serverMajor := uint8((f.header.ProtocolID >> 24) & 0xFF)
				serverMinor := uint8((f.header.ProtocolID >> 16) & 0xFF)

				if clientMajor != serverMajor || clientMinor != serverMinor {
					panic("Server is running an unsupported protocol version.")
				}

				if f.header.BodySize > SpyserverMaxCommandBodySize {
					panic("The server is probably buggy.")
				}

				f.bodyBuffer = make([]uint8, f.header.BodySize)
			}
		}

		if f.parserPhase == ParserReadingData {
			consumed = f.parseBody(buffer)
			buffer = buffer[consumed:]

			if f.parserPhase == ParserAcquiringHeader {
				if f.header.MessageType != MsgTypeDeviceInfo && f.header.MessageType != MsgTypeClientSync {
					gap := f.header.SequenceNumber - f.lastSequenceNumber - 1
					f.lastSequenceNumber = f.header.SequenceNumber
					f.droppedBuffers += gap
					if gap > 0 {
						log.Printf("Lost %d frames from spyserver!\n", gap)
					}
				}
				f.handleNewMessage()
			}
		}
	}
}
func (f *Spyserver) parseHeader(buffer []uint8) uint32 {
	consumed := uint32(0)

	for len(buffer) > 0 {
		toWrite := min(MessageHeaderSize - f.parserPosition, uint32(len(buffer)))
		for i := f.parserPosition; i < toWrite; i++ {
			f.headerBuffer[i] = buffer[i-f.parserPosition]
		}
		buffer = buffer[toWrite:]
		consumed += toWrite
		f.parserPosition += toWrite

		if f.parserPosition == MessageHeaderSize {
			f.parserPosition = 0
			buf := bytes.NewReader(f.headerBuffer)
			err := binary.Read(buf, binary.LittleEndian, &f.header)
			if err != nil {
				panic(err)
			}

			if f.header.BodySize > 0 {
				f.parserPhase = ParserReadingData
			}

			return consumed
		}
	}

	return consumed
}
func (f *Spyserver) parseBody(buffer []uint8) uint32 {
	consumed := uint32(0)

	for len(buffer) > 0 {
		toWrite := min(f.header.BodySize - f.parserPosition, uint32(len(buffer)))
		for i := f.parserPosition; i < toWrite; i++ {
			buffer = buffer[toWrite:]
		}
		buffer = buffer[toWrite:]
		consumed += toWrite
		f.parserPosition += toWrite

		if f.parserPosition == f.header.BodySize {
			f.parserPosition = 0
			f.parserPhase = ParserAcquiringHeader
			return consumed
		}
	}

	return consumed
}
func (f *Spyserver) processDeviceInfo() {

}
func (f *Spyserver) processClientSync() {

}
func (f *Spyserver) processUInt8Samples() {

}
func (f *Spyserver) processInt16Samples() {

}
func (f *Spyserver) processFloatSamples() {

}
func (f *Spyserver) processUInt8FFT() {

}
func (f * Spyserver) handleNewMessage() {

}
func (f *Spyserver) setStreamState() {

}
func (f *Spyserver) threadLoop() {
	f.parserPhase = ParserAcquiringHeader
	f.parserPosition = 0

	buffer := make([]uint8, 64 * 1024)

	for f.routineRunning && !f.terminated {
		if f.terminated || !f.routineRunning {
			break
		}

		n, err := f.client.Read(buffer)
		if err != nil {
			log.Println("Error receiving data: ", err)
			break
		}
		if n > 0 {
			var sl = buffer[:n]
			f.parseMessage(sl)
		}
	}

	f.routineRunning = false
	f.cleanup()
}
// endregion
// region Public Methods
func (f *Spyserver) GetName() string {
	return DeviceName[DeviceInvalid]
}
func (f *Spyserver) Start() {

}
func (f *Spyserver) Stop() {

}
func (f *Spyserver) Connect() {
	if f.routineRunning {
		return
	}

	log.Println("Trying to connect")
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", f.hostname, f.port))
	if err != nil {
		panic(err)
	}

	f.client = conn
	f.IsConnected = true

	f.sayHello()
	f.cleanup()

	f.terminated = false
	f.gotSyncInfo = false
	f.gotDeviceInfo = false

	hasError := false
	errorMsg := ""

	go f.threadLoop()

	for i := 0; i < 1000 && !hasError; i ++ {
		if f.gotDeviceInfo {
			if f.deviceInfo.DeviceType == DeviceInvalid {
				errorMsg = "Server is up but no device is available"
				hasError = true
				break
			}

			if f.gotSyncInfo {
				f.onConnect()
				return
			}
		}
		time.Sleep(time.Millisecond)
	}

	f.Disconnect()
	if hasError {
		hasError = false
		panic(errorMsg)
	}

	panic("Server didn't send the device capability and synchronization info.")
}
func (f * Spyserver) Disconnect() {
	f.terminated = true
	if f.IsConnected {
		f.client.Close()
	}

	f.routineRunning = false

	f.cleanup()
}
func (f *Spyserver) GetSampleRate() uint32 {
	return 0
}
func (f *Spyserver) SetSampleRate(sampleRate uint32) uint32 {
	return 0
}
func (f *Spyserver) GetCenterFrequency() uint32 {
	return 0
}
func (f *Spyserver) SetCenterFrequency(centerFrequency uint32) uint32 {
	return 0
}
func (f *Spyserver) SetCallback(cb *CallbackBase) {
	f.callback = cb
}
// endregion