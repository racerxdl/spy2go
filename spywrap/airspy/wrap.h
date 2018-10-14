#include "airspy.h"

typedef void (*callback)(void *, airspy_transfer *);
int cbProxy(void *v, airspy_transfer* d);

typedef struct {
    struct airspy_device* device;
    int result;
} airspy_open_result_t;

airspy_open_result_t* openDevice();
airspy_open_result_t* openDeviceBySerial(uint64_t serial_number);
uint64_t serialNumber(uint32_t *data);
uint64_t partNumber(uint32_t *data);
void freeOpenResult(airspy_open_result_t*);
int airspyStart(struct airspy_device* device, void *data);