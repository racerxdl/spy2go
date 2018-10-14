package spywrap

const (
	AirspySuccess                 = 0
	AirspyTrue                    = 1
	AirspyErrorInvalidParam       = -2
	AirspyErrorNotFound           = -5
	AirspyErrorBusy               = -6
	AirspyErrorNoMem              = -11
	AirspyErrorLibusb             = -1000
	AirspyErrorThread             = -1001
	AirspyErrorStreamingThreadErr = -1002
	AirspyErrorStreamingStopped   = -1003
	AirspyErrorOther              = -9999
)

const (
	AirspyBoardIdProtoAirspy = 0
	AirspyBoardIdInvalid     = 0xFF
)

const (
	AirspySampleFloat32Iq   = 0 /* 2 * 32bit float per sample */
	AirspySampleFloat32Real = 1 /* 1 * 32bit float per sample */
	AirspySampleInt16Iq     = 2 /* 2 * 16bit int per sample */
	AirspySampleInt16Real   = 3 /* 1 * 16bit int per sample */
	AirspySampleUint16Real  = 4 /* 1 * 16bit unsigned int per sample */
	AirspySampleRaw         = 5 /* Raw packed samples from the device */
	AirspySampleEnd         = 6 /* Number of supported sample types */
)
