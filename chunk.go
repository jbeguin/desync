package desync

import (
	"errors"
	"fmt"
)

// Chunk holds chunk data packaged according to flags isCompress and isEncrypt.
// It compress/uncompress encrypt/decrypt on demand
type Chunk struct {
	packagedData []byte
	id           ChunkID
	idCalculated bool
	isCompress   bool
	isEncrypt    bool
}

// NewChunkFromUncompressed creates a new chunk from uncompressed and unencrypted data.
func NewChunkFromUncompressed(b []byte) *Chunk {
	return &Chunk{packagedData: b, isCompress: false, isEncrypt: false}
}

// NewChunkWithID creates a new chunk from already packaged data
// according to flags isCompress and isEncrypt.
// If isEncrypt, key must be there.
// It also expects an ID and validates that it matches
// the uncompressed and unencrypted data unless skipVerify is true. If called with packaged data
// (either compress/uncompress encrypt/decrypt) it'll depackage it for the ID validation.
func NewChunkWithID(id ChunkID, packagedData, key []byte, skipVerify, isCompress, isEncrypt bool) (*Chunk, error) {
	c := &Chunk{id: id, packagedData: packagedData, isCompress: isCompress, isEncrypt: isEncrypt}
	if skipVerify {
		c.idCalculated = true // Pretend this was calculated. No need to re-calc later
		return c, nil
	}
	sum := c.ID(key)
	if sum != id {
		fmt.Printf("NewChunkWithID ChunkInvalid b = %s\n", packagedData)
		return nil, ChunkInvalid{ID: id, Sum: sum}
	}
	return c, nil
}

// Return packaged data from internal data state according to destination flags dst_compress and dst_encrypt
func (c *Chunk) GetPackagedData(key []byte, dst_compress, dst_encrypt bool) ([]byte, error) {
	if len(c.packagedData) == 0 {
		Log.Error("no data in chunk\n")
		return nil, errors.New("no data in chunk")
	}
	if c.isCompress == dst_compress && c.isEncrypt == dst_encrypt {
		return c.packagedData, nil
	}
	if !c.isEncrypt && c.isEncrypt == dst_encrypt {
		if c.isCompress {
			return Decompress(nil, c.packagedData)
		} else {
			return Compress(c.packagedData)
		}
	}
	if key == nil {
		Log.Error("no key for chunk encryption/decryption")
		return nil, errors.New("no key for chunk encryption/decryption")
	}
	if c.isEncrypt == dst_encrypt {
		var data []byte
		data, err := Decrypt(c.packagedData, key)
		if err != nil {
			return nil, err
		}
		if c.isCompress {
			data, err = Decompress(nil, data)
		} else {
			data, err = Compress(data)
		}
		if err != nil {
			return nil, err
		}
		return Encrypt(data, key)
	}
	if c.isCompress == dst_compress {
		if c.isEncrypt {
			return Decrypt(c.packagedData, key)
		} else {
			return Encrypt(c.packagedData, key)
		}
	}
	if c.isEncrypt {
		d, err := Decrypt(c.packagedData, key)
		if err != nil {
			return nil, err
		}
		if c.isCompress {
			return Decompress(nil, d)
		} else {
			return Compress(d)
		}
	}
	if c.isCompress {
		comp, err := Decompress(nil, c.packagedData)
		if err != nil {
			return nil, err
		}
		return Encrypt(comp, key)
	}
	comp, err := Compress(c.packagedData)
	if err != nil {
		return nil, err
	}
	return Encrypt(comp, key)
}

// ID returns the checksum/ID of the uncompressed chunk data. The ID is stored
// after the first call and doesn't need to be re-calculated. Note that calculating
// the ID may mean decompressing and/or decrypting the data first.
func (c *Chunk) ID(key []byte) ChunkID {
	if c.idCalculated {
		return c.id
	}
	b, err := c.GetPackagedData(key, false, false)

	if err != nil {
		return ChunkID{}
	}
	c.id = Digest.Sum(b)
	c.idCalculated = true
	return c.id
}
