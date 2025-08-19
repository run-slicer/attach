//go:build windows
// +build windows

#include <windows.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>

#define OPEN_JVM_ERROR 200
#define GET_ENQUEUE_FUNCTION_ERROR 201
#define CREATE_REMOTE_THREAD_ERROR 202
#define WAIT_TIMEOUT_ERROR 203
#define CODE_SIZE (SIZE_T) 1024
#define MAX_ARGUMENT 1024

typedef HMODULE (WINAPI *GetModuleHandle_t)(LPCTSTR);
typedef FARPROC (WINAPI *GetProcAddress_t)(HMODULE, LPCSTR);
typedef int (__stdcall *JVM_EnqueueOperation_t)(char *, char *, char *, char *, char *);

typedef struct {
    GetModuleHandle_t GetModuleHandleA;
    GetProcAddress_t GetProcAddress;
    char library[32];
    char command[32];
    char commandFallback[32];
    char pipe[MAX_PATH];
    char argument[4][MAX_ARGUMENT];
} EnqueueOperation;

#pragma check_stack(off)

/**
 * Executes the attachment on the remote thread. This method is executed on the target JVM and must not reference any addresses unknown to that address.
 */
DWORD WINAPI execute_remote_attach(LPVOID argument) {
    EnqueueOperation *operation = (EnqueueOperation *) argument;
    HMODULE library = operation->GetModuleHandleA(operation->library);
    if (library == NULL) {
        return OPEN_JVM_ERROR;
    }
    JVM_EnqueueOperation_t JVM_EnqueueOperation = (JVM_EnqueueOperation_t) operation->GetProcAddress(library, operation->command);
    if (JVM_EnqueueOperation == NULL) {
        JVM_EnqueueOperation = (JVM_EnqueueOperation_t) operation->GetProcAddress(library, operation->commandFallback);
    }
    if (JVM_EnqueueOperation == NULL) {
        return GET_ENQUEUE_FUNCTION_ERROR;
    }
    return (DWORD) JVM_EnqueueOperation(operation->argument[0],
                                        operation->argument[1],
                                        operation->argument[2],
                                        operation->argument[3],
                                        operation->pipe);
}

#pragma check_stack

/**
 * Allocates the code to execute on the remote machine.
 */
static LPVOID allocate_remote_code(HANDLE process) {
    LPVOID code = VirtualAllocEx(process, NULL, CODE_SIZE, MEM_COMMIT, PAGE_EXECUTE_READWRITE);
    if (code == NULL) {
        return NULL;
    } else if (!WriteProcessMemory(process, code, execute_remote_attach, CODE_SIZE, NULL)) {
        VirtualFreeEx(process, code, 0, MEM_RELEASE);
        return NULL;
    } else {
        return code;
    }
}

/**
 * Allocates the argument to the remote execution.
 */
static LPVOID allocate_remote_argument(HANDLE process, LPCSTR pipe, LPCSTR argument0, LPCSTR argument1, LPCSTR argument2, LPCSTR argument3) {
    if (strlen(pipe) >= MAX_PATH
            || (argument0 != NULL && strlen(argument0) >= MAX_ARGUMENT)
            || (argument1 != NULL && strlen(argument1) >= MAX_ARGUMENT)
            || (argument2 != NULL && strlen(argument2) >= MAX_ARGUMENT)
            || (argument3 != NULL && strlen(argument3) >= MAX_ARGUMENT)) {
        return NULL;
    }
    EnqueueOperation operation;
    operation.GetModuleHandleA = GetModuleHandleA;
    operation.GetProcAddress = GetProcAddress;
    strcpy(operation.library, "jvm");
    strcpy(operation.command, "JVM_EnqueueOperation");
    strcpy(operation.commandFallback, "_JVM_EnqueueOperation@20");
    strcpy(operation.pipe, pipe);
    strcpy(operation.argument[0], argument0 == NULL ? "" : argument0);
    strcpy(operation.argument[1], argument1 == NULL ? "" : argument1);
    strcpy(operation.argument[2], argument2 == NULL ? "" : argument2);
    strcpy(operation.argument[3], argument3 == NULL ? "" : argument3);
    LPVOID allocation = VirtualAllocEx(process, NULL, sizeof(EnqueueOperation), MEM_COMMIT, PAGE_READWRITE);
    if (allocation == NULL) {
        return NULL;
    } else if (!WriteProcessMemory(process, allocation, &operation, sizeof(operation), NULL)) {
        VirtualFreeEx(process, allocation, 0, MEM_RELEASE);
        return NULL;
    } else {
        return allocation;
    }
}

/**
 * Attaches to a JVM process using process injection.
 * Returns 0 on success, non-zero on error.
 */
int attach_to_jvm(DWORD pid, const char* pipe_name, const char* arg0, const char* arg1, const char* arg2, const char* arg3) {
    HANDLE process = OpenProcess(PROCESS_CREATE_THREAD | PROCESS_QUERY_INFORMATION | PROCESS_VM_OPERATION | PROCESS_VM_WRITE | PROCESS_VM_READ, FALSE, pid);
    if (process == NULL) {
        return GetLastError();
    }

    LPVOID remote_code = allocate_remote_code(process);
    if (remote_code == NULL) {
        CloseHandle(process);
        return GetLastError();
    }

    LPVOID remote_argument = allocate_remote_argument(process, pipe_name, arg0, arg1, arg2, arg3);
    if (remote_argument == NULL) {
        VirtualFreeEx(process, remote_code, 0, MEM_RELEASE);
        CloseHandle(process);
        return GetLastError();
    }

    HANDLE remote_thread = CreateRemoteThread(process, NULL, 0, (LPTHREAD_START_ROUTINE) remote_code, remote_argument, 0, NULL);
    if (remote_thread == NULL) {
        VirtualFreeEx(process, remote_argument, 0, MEM_RELEASE);
        VirtualFreeEx(process, remote_code, 0, MEM_RELEASE);
        CloseHandle(process);
        return CREATE_REMOTE_THREAD_ERROR;
    }

    DWORD wait_result = WaitForSingleObject(remote_thread, 10000); // 10 second timeout
    DWORD exit_code = 0;
    if (wait_result == WAIT_OBJECT_0) {
        GetExitCodeThread(remote_thread, &exit_code);
    } else {
        exit_code = WAIT_TIMEOUT_ERROR;
    }

    CloseHandle(remote_thread);
    VirtualFreeEx(process, remote_argument, 0, MEM_RELEASE);
    VirtualFreeEx(process, remote_code, 0, MEM_RELEASE);
    CloseHandle(process);

    return exit_code;
}

/**
 * Creates a named pipe with the given name.
 * Returns the pipe handle on success, INVALID_HANDLE_VALUE on error.
 */
HANDLE create_attach_pipe(const char* pipe_name) {
    char full_pipe_name[MAX_PATH];
    snprintf(full_pipe_name, sizeof(full_pipe_name), "\\\\.\\pipe\\%s", pipe_name);

    SECURITY_DESCRIPTOR sd;
    InitializeSecurityDescriptor(&sd, SECURITY_DESCRIPTOR_REVISION);
    SetSecurityDescriptorDacl(&sd, TRUE, NULL, FALSE);

    SECURITY_ATTRIBUTES sa;
    sa.nLength = sizeof(SECURITY_ATTRIBUTES);
    sa.lpSecurityDescriptor = &sd;
    sa.bInheritHandle = FALSE;

    HANDLE pipe = CreateNamedPipeA(
        full_pipe_name,
        PIPE_ACCESS_DUPLEX | FILE_FLAG_FIRST_PIPE_INSTANCE,
        PIPE_TYPE_BYTE | PIPE_READMODE_BYTE | PIPE_WAIT | PIPE_REJECT_REMOTE_CLIENTS,
        1,      // max instances
        4096,   // output buffer size
        8192,   // input buffer size
        0,      // default timeout
        &sa     // security attributes
    );

    return pipe;
}