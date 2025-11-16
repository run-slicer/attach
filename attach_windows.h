#include <windows.h>

typedef struct {
    int success;
    DWORD error_code;
    char error_msg[256];
} attach_result_t;

typedef struct {
    char* data;
    int capacity;
    int length;
} response_buffer_t;

int attach(int pid, int argc, char** argv, response_buffer_t* respBuf, attach_result_t* result);
