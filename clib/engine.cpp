#include "engine.h"
#include <openssl/sha.h>
#include <vector>
#include <string>
#include <cstring>
#include <cstdlib>
#include <cstdio>

int merkle_root(const char** inputs, const size_t* lengths, size_t n, unsigned char** out_root, char* errbuf, int errbuf_len) {
    if (n == 0) {
        snprintf(errbuf, errbuf_len, "Empty input");
        return 1;
    }

    std::vector<std::vector<unsigned char>> hashes;
    hashes.reserve(n);

    for (size_t i = 0; i < n; ++i) {
        unsigned char hash[SHA256_SIZE];
        SHA256(reinterpret_cast<const unsigned char*>(inputs[i]), lengths[i], hash);
        hashes.push_back(std::vector<unsigned char>(hash, hash + SHA256_SIZE));
    }

    // Построение дерева
    while (hashes.size() > 1) {
        std::vector<std::vector<unsigned char>> next;
        for (size_t i = 0; i < hashes.size(); i += 2) {
            std::vector<unsigned char> concat;
            concat.insert(concat.end(), hashes[i].begin(), hashes[i].end());
            if (i + 1 < hashes.size())
                concat.insert(concat.end(), hashes[i+1].begin(), hashes[i+1].end());
            else
                concat.insert(concat.end(), hashes[i].begin(), hashes[i].end());
            unsigned char hash[SHA256_SIZE];
            SHA256(concat.data(), concat.size(), hash);
            next.push_back(std::vector<unsigned char>(hash, hash + SHA256_SIZE));
        }
        hashes = next;
    }

    *out_root = (unsigned char*)malloc(SHA256_SIZE);
    if (!*out_root) {
        snprintf(errbuf, errbuf_len, "malloc failed");
        return 2;
    }
    memcpy(*out_root, hashes[0].data(), SHA256_SIZE);
    return 0;
}

void free_root(unsigned char* root) {
    free(root);
}
