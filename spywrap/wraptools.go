package spywrap

import "C"

func GetAirspyError(err int) string {
	return Airspy_error_name(Enum_SS_airspy_error(err))
}

func CharStringToString(data []byte) string {
	var length = 0
	for i := 0; i < len(data); i++ {
		length = i
		if data[i] == 0x00 {
			break
		}
	}
	return string(data[:length])
}
