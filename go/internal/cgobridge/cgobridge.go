package cgobridge

/*
#cgo LDFLAGS: -L../../clib/build -lengine -lcrypto
#cgo CFLAGS: -I${SRCDIR}/../../../clib
#include <stdlib.h>
#include "engine.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

// MerkleRoot вызывает C++ функцию merkle_root и возвращает SHA256 root
func MerkleRoot(messages [][]byte) ([]byte, error) {
    n := len(messages)
    if n == 0 {
        return nil, errors.New("empty messages")
    }

    // Подготовка массивов C - ВСЕ данные должны быть в C памяти
    cInputs := make([]*C.char, n)
    cLengths := make([]C.size_t, n)
    
    // Слайс для отслеживания памяти, которую нужно освободить
    allocated := make([]unsafe.Pointer, 0, n)

    defer func() {
        // Освобождаем всю выделенную C память
        for _, ptr := range allocated {
            C.free(ptr)
        }
    }()

    for i, msg := range messages {
        if len(msg) == 0 {
            // Для пустого сообщения создаем нулевой указатель
            cInputs[i] = nil
            cLengths[i] = 0
        } else {
            // Копируем данные в C память
            cData := C.CBytes(msg)
            cInputs[i] = (*C.char)(cData)
            cLengths[i] = C.size_t(len(msg))
            allocated = append(allocated, cData)
        }
    }

    var outRoot *C.uchar
    errbuf := make([]C.char, C.ENGINE_ERRBUF_SIZE)

    res := C.merkle_root(
        &cInputs[0],
        &cLengths[0],
        C.size_t(n),
        &outRoot,
        &errbuf[0],
        C.int(C.ENGINE_ERRBUF_SIZE),
    )

    if res != 0 {
        return nil, errors.New(C.GoString(&errbuf[0]))
    }

    // Копируем результат в Go
    root := C.GoBytes(unsafe.Pointer(outRoot), C.SHA256_SIZE)
    C.free_root(outRoot)

    return root, nil
}