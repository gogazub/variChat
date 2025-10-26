package cgobridge

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerkleRoot(t *testing.T) {
	tests := []struct {
		name     string
		messages [][]byte
		wantErr  bool
		checkFn  func(t *testing.T, result []byte)
	}{
		{
			name:     "single message",
			messages: [][]byte{[]byte("hello")},
			wantErr:  false,
			checkFn: func(t *testing.T, result []byte) {
				expected := sha256.Sum256([]byte("hello"))
				assert.Equal(t, expected[:], result, "should match direct SHA256")
			},
		},
		{
			name: "two messages",
			messages: [][]byte{
				[]byte("message1"),
				[]byte("message2"),
			},
			wantErr: false,
			checkFn: func(t *testing.T, result []byte) {
				assert.Len(t, result, 32, "should be 32 bytes SHA256")
			},
		},
		{
			name: "three messages",
			messages: [][]byte{
				[]byte("a"),
				[]byte("b"),
				[]byte("c"),
			},
			wantErr: false,
		},
		{
			name: "empty message in middle",
			messages: [][]byte{
				[]byte("start"),
				{},
				[]byte("end"),
			},
			wantErr: false,
		},
		{
			name:     "empty messages array",
			messages: [][]byte{},
			wantErr:  true,
		},
		{
			name: "multiple empty messages",
			messages: [][]byte{
				{},
				{},
				{},
			},
			wantErr: false,
		},
		{
			name: "binary data",
			messages: [][]byte{
				{0x00, 0x01, 0x02, 0x03},
				{0xff, 0xfe, 0xfd},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MerkleRoot(tt.messages)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result, 32, "Merkle root should be 32 bytes")

				if tt.checkFn != nil {
					tt.checkFn(t, result)
				}

				assert.False(t, allZero(result), "Merkle root should not be all zeros")
			}
		})
	}
}

func TestMerkleRootConsistency(t *testing.T) {
	messages1 := [][]byte{
		[]byte("test1"),
		[]byte("test2"),
		[]byte("test3"),
	}

	messages2 := [][]byte{
		[]byte("test1"),
		[]byte("test2"), 
		[]byte("test3"),
	}

	result1, err1 := MerkleRoot(messages1)
	require.NoError(t, err1)

	result2, err2 := MerkleRoot(messages2)  
	require.NoError(t, err2)

	assert.Equal(t, result1, result2, "same input should produce same Merkle root")
}

func TestMerkleRootDifferentInputs(t *testing.T) {
	messages1 := [][]byte{[]byte("hello")}
	messages2 := [][]byte{[]byte("world")}

	result1, err1 := MerkleRoot(messages1)
	require.NoError(t, err1)

	result2, err2 := MerkleRoot(messages2)
	require.NoError(t, err2)

	assert.NotEqual(t, result1, result2, "different input should produce different Merkle roots")
}

func TestMerkleRootOrderMatters(t *testing.T) {
	messages1 := [][]byte{
		[]byte("first"),
		[]byte("second"),
	}

	messages2 := [][]byte{
		[]byte("second"), 
		[]byte("first"),
	}

	result1, err1 := MerkleRoot(messages1)
	require.NoError(t, err1)

	result2, err2 := MerkleRoot(messages2)
	require.NoError(t, err2)

	assert.NotEqual(t, result1, result2, "order of messages should affect Merkle root")
}

func BenchmarkMerkleRoot(b *testing.B) {
	messages := [][]byte{
		[]byte("message 1"),
		[]byte("message 2"),
		[]byte("message 3"),
		[]byte("message 4"),
		[]byte("message 5"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MerkleRoot(messages)
		if err != nil {
			b.Fatalf("MerkleRoot failed: %v", err)
		}
	}
}

func TestMerkleRootLargeInput(t *testing.T) {
	largeMessage := make([]byte, 10000) // 10KB
	for i := range largeMessage {
		largeMessage[i] = byte(i % 256)
	}

	messages := [][]byte{
		largeMessage,
		[]byte("small message"),
	}

	result, err := MerkleRoot(messages)
	require.NoError(t, err)
	assert.Len(t, result, 32)
}

func allZero(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}

func ExampleMerkleRoot() {
	messages := [][]byte{
		[]byte("transaction1"),
		[]byte("transaction2"),
		[]byte("transaction3"),
	}

	root, err := MerkleRoot(messages)
	if err != nil {
		return
	}

	hexRoot := hex.EncodeToString(root)
	println("Merkle Root:", hexRoot)
}