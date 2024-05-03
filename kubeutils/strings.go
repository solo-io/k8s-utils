package kubeutils

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"hash"
	"sync"
)

// This is an extrememly important number for these shortened names.
// It signifies the spot of separation from the original name and the hash.
// It is used for many string building and parsing operations below.
const magicNumber = 31
const totalSize = magicNumber*2 + 1
const encodedMd5 = 2 * md5.Size

const separator = '-'

// We can short-circuit the comparison if the first 31 characters are not equal.
// Otherweise we need to compare the shortened version of the strings.
func ShortenedEquals(shortened, standard string) bool {

	// If the standard string is less than 63 characters, we can just compare the strings.
	if len(standard) <= totalSize {
		return shortened == standard
	}

	// If the shortened string is less than or equal to 32 characters, we can just compare the strings.
	// Also if it's less than 32 the below checks may crash.
	if len(shortened) <= magicNumber+1 {
		return shortened == standard
	}

	// Check the first 31 characters, if they're not equal we can exit early.
	if shortened[:magicNumber] != standard[:magicNumber] {
		return false
	}

	// If 32nd character of the shortened string is not a '-' or the 32nd character of the standard string is not a '-'
	// we can exit early.
	// In theory this shouldn't be necessary, but this label can technically be modified by the user,
	// so it's safer to double check.
	if shortened[magicNumber] != separator {
		return false
	}

	// Check the last 32 characters of the shortened string against the hash of the standard string.
	hashed := hashName(standard)
	return shortened[magicNumber+1:] == string(hashed[:magicNumber])
}

// shortenName is extrememly inefficient with it's allocation of slices for hashing.
// We can re-use the arrays to avoid this allocation. However, this code may be called
// from multiple go-routines simultaneously so we must house these objects in sync.Pools

// Pool of MD5 hashers to avoid allocation.
var md5HasherPool = sync.Pool{
	New: func() interface{} {
		return md5.New()
	},
}

// Pool of string builders to avoid allocation.
var byteBufferPool = sync.Pool{
	New: func() interface{} {
		b := &bytes.Buffer{}
		b.Grow(totalSize)
		return b
	},
}

// hashName returns a hash of the input string in base 16 format
// This function is optimized for speed and memory usage.
// It should aboid nearly all allocations by re-using the same buffers whenever possible.
func hashName(name string) [encodedMd5]byte {
	hasher := md5HasherPool.Get().(hash.Hash)
	hasher.Reset()
	hasher.Write([]byte(name))
	hashArray := [md5.Size]byte{}
	hash := hasher.Sum(hashArray[:0])
	// Cannot use hex.EncodedLen() here because it's a func, but it just returns 2 * len(src)
	hashBufferArray := [encodedMd5]byte{}
	hex.Encode(hashBufferArray[:], hash)
	md5HasherPool.Put(hasher)
	return hashBufferArray
}

// shortenName returns a shortened version of the input string.
// It is based on the `kubeutils.SanitizeNameV2` function, but it
// just does the shortening part.
func ShortenName(name string) string {
	if len(name) > totalSize {
		hash := hashName(name)
		builder := byteBufferPool.Get().(*bytes.Buffer)
		builder.Reset()
		builder.Grow(totalSize)
		builder.WriteString(name[:magicNumber])
		builder.WriteRune(separator)
		builder.Write(hash[:magicNumber])
		name = builder.String()
		byteBufferPool.Put(builder)
	}
	return name
}
