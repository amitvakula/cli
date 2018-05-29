import os
from ctypes import *

# Build with: ./make.sh go build -buildmode=c-shared -o build/fw.dylib fw.go
SRC_DIR = os.path.abspath(os.path.dirname(__file__))
LEGACY_LIB_PATH = os.path.join(SRC_DIR, '../build/fw.dylib')

#typedef struct { void *data; GoInt len; GoInt cap; } GoSlice;
class GoSlice(Structure):
    _fields_ = [
        ("data", POINTER(c_void_p)), 
        ("len", c_longlong), 
        ("cap", c_longlong)
    ]

#typedef struct { const char *p; GoInt n; } GoString;
class GoString(Structure):
    _fields_ = [
        ("p", c_char_p),
        ("n", c_longlong)
    ]

def args_to_slice(args):
    """ Convert a list of args to a GoSlice structure.

    Arguments:
        args (list): The list of arguments to convert

    Returns:
        GoSlice: The converted arguments
    """
    argc = len(args)
    str_list = []
    for i in range(argc):
        str_list.append(GoString(args[i].encode('utf-8'), len(args[i])))

    str_list = (GoString * argc)(*str_list)
    return GoSlice(cast(str_list, POINTER(c_void_p)), argc, argc)

def invoke_command(args):
    """ Invoke a legacy command by calling the go library.

    Arguments:
        args (list): The list of arguments to pass

    Returns:
        int: The return code
    """
    lib = cdll.LoadLibrary(LEGACY_LIB_PATH)
    lib.InvokeCommand.argtypes = [GoSlice]

    return lib.InvokeCommand(args_to_slice(args))



