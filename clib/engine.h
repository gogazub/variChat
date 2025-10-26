#pragma once
#ifdef __cplusplus
extern "C" {
#endif

#include <stddef.h>

#define ENGINE_ERRBUF_SIZE 256

// Размер SHA256 хеша
#define SHA256_SIZE 32

// Выделяет Merkle-root из массива сообщений
// inputs: массив указателей на байтовые строки
// lengths: длины каждой строки
// n: количество сообщений
// out_root: malloc'ed 32 байта root (нужно free_root)
// errbuf: buffer для ошибок (ENGINE_ERRBUF_SIZE)
// Возвращает 0 если success, иначе !=0
int merkle_root(const char** inputs, const size_t* lengths, size_t n, unsigned char** out_root, char* errbuf, int errbuf_len);

// Освобождение root
void free_root(unsigned char* root);

#ifdef __cplusplus
}
#endif
