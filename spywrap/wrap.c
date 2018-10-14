#include "airspy/wrap.h"

#include <stdlib.h>

/*
    Ugly workaround for SWIG 3.0 bug with pointer to pointer
*/

airspy_open_result_t* openDevice() {
    airspy_open_result_t *res = malloc(sizeof(airspy_open_result_t));
    res->result = airspy_open(&res->device);
    return res;
}

airspy_open_result_t* openDeviceBySerial(uint64_t serial_number) {
    airspy_open_result_t *res = malloc(sizeof(airspy_open_result_t));
    res->result = airspy_open_sn(&res->device, serial_number);
    return res;
}

void freeOpenResult(airspy_open_result_t *d) {
    free(d);
}

uint64_t serialNumber(uint32_t *data) {
    return (uint64_t)data[3] << 32 | data[2];
}

uint64_t partNumber(uint32_t *data) {
    return (uint64_t)data[1] << 32 | data[0];
}
int cbProxyNative(airspy_transfer* transfer) {
    return cbProxy(transfer->ctx, transfer);
}

int airspyStart(struct airspy_device* device, void *data) {
    return airspy_start_rx(device, cbProxyNative, data);
}
