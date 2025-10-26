package cgobridge

/*
#cgo LDFLAGS: -L${SRCDIR}/../../../clib -lengine
#include <stdlib.h>
extern int add(int a, int b);
extern const char* hello();
*/
import "C"


func Add(a, b int) int {
    return int(C.add(C.int(a), C.int(b)))
}

func Hello() string{
	return C.GoString(C.hello())
}