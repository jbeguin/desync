package desync

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func TestHTTPHandlerReadWrite(t *testing.T) {
	// Setup a temporary store used as upstream store that the HTTP server
	// pulls from
	store, err := ioutil.TempDir("", "store")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(store)

	upstream, err := NewLocalStore(store, StoreOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Start a read-write capable server and a read-only server
	rw := httptest.NewServer(NewHTTPHandler(upstream, true, false, false, false, nil, ""))
	defer rw.Close()
	ro := httptest.NewServer(NewHTTPHandler(upstream, false, false, false, false, nil, ""))
	defer ro.Close()

	// Initialize HTTP chunks stores, one RW and the other RO
	rwStoreURL, _ := url.Parse(rw.URL)
	rwStore, err := NewRemoteHTTPStore(rwStoreURL, StoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	roStoreURL, _ := url.Parse(ro.URL)
	roStore, err := NewRemoteHTTPStore(roStoreURL, StoreOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Make up some data and store it in the RW store
	dataIn := []byte("some data")
	chunkIn := NewChunkFromUncompressed(dataIn)
	id := chunkIn.ID(nil)
	if err := rwStore.StoreChunk(chunkIn); err != nil {
		t.Fatal(err)
	}

	// Check it's in the store
	hashChunk, err := rwStore.HasChunk(id)
	if err != nil {
		t.Fatal(err)
	}
	if !hashChunk {
		t.Fatal("chunk not found in store")
	}

	// Let's try to send some data to the RO store, that should fail
	if err := roStore.StoreChunk(chunkIn); err == nil {
		t.Fatal("expected error writing to read-only chunkstore")
	}
}

func TestHTTPHandlerCompression(t *testing.T) {
	// Setup a temporary store used as upstream store that the HTTP server
	// pulls from
	store, err := ioutil.TempDir("", "store")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(store)

	upstream, err := NewLocalStore(store, StoreOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Start a server that uses compression, and one that serves uncompressed chunks
	co := httptest.NewServer(NewHTTPHandler(upstream, true, false, false, false, nil, ""))
	defer co.Close()
	un := httptest.NewServer(NewHTTPHandler(upstream, true, false, true, false, nil, ""))
	defer un.Close()

	// Initialize HTTP chunks stores, one RW and the other RO. Also make one that's
	// trying to get compressed data from a HTTP store that serves only uncompressed.
	coStoreURL, _ := url.Parse(co.URL)
	coStore, err := NewRemoteHTTPStore(coStoreURL, StoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	unStoreURL, _ := url.Parse(un.URL)
	unStore, err := NewRemoteHTTPStore(unStoreURL, StoreOptions{Uncompressed: true})
	if err != nil {
		t.Fatal(err)
	}
	invalidStore, err := NewRemoteHTTPStore(unStoreURL, StoreOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Make up some data and store it in the RW store
	dataIn := []byte("some data")
	chunkIn := NewChunkFromUncompressed(dataIn)
	id := chunkIn.ID(nil)

	// Try to get compressed chunks from a store that only serves uncompressed chunks
	if _, err := invalidStore.GetChunk(id); err == nil {
		t.Fatal("expected failure trying to get compressed chunks from uncompressed http store")
	}

	if err := coStore.StoreChunk(chunkIn); err != nil {
		t.Fatal(err)
	}

	// Check it's in the store when looking for compressed chunks
	coExists, err := coStore.HasChunk(id)
	if err != nil {
		t.Fatal(err)
	}
	if !coExists {
		t.Fatal("chunk not found in store")
	}

	// It's also visible when looking for uncompressed data
	unExists, err := unStore.HasChunk(id)
	if err != nil {
		t.Fatal(err)
	}
	if !unExists {
		t.Fatal("chunk not found in store")
	}

	// Send it uncompressed
	if err := unStore.StoreChunk(chunkIn); err != nil {
		t.Fatal(err)
	}

	// Try to get the uncompressed chunk
	if _, err := unStore.GetChunk(id); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPHandlerPackaging(t *testing.T) {
	for s1compress := 0; s1compress < 2; s1compress++ {
		for s1encrypt := 0; s1encrypt < 2; s1encrypt++ {
			for s2compress := 0; s2compress < 2; s2compress++ {
				for s2encrypt := 0; s2encrypt < 2; s2encrypt++ {
					testHTTPHandlerPackaging(t, s1compress == 1, s1encrypt == 1, s2compress == 1, s2encrypt == 1, []byte("12345678901234567890123456789012"))
					fmt.Printf("s1compress %t, s1encrypt %t, s2compress %t, s2encrypt %t\n", s1compress == 1, s1encrypt == 1, s2compress == 1, s2encrypt == 1)
				}
			}
		}
	}
}

func testHTTPHandlerPackaging(t *testing.T, s1compress, s1encrypt, s2compress, s2encrypt bool, key []byte) {
	// Setup a temporary store used as upstream store that the HTTP server
	// pulls from
	store, err := ioutil.TempDir("", "store")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(store)

	upstream, err := NewLocalStore(store, StoreOptions{EncryptionKey: []byte("12345678901234567890123456789012")})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("s1compress %t, s1encrypt %t, s2compress %t, s2encrypt %t\n", s1compress, s1encrypt, s2compress, s2encrypt)
	// Start a server that uses compression, and one that serves uncompressed chunks
	co := httptest.NewServer(NewHTTPHandler(upstream, true, false, s1compress, s1encrypt, key, ""))
	defer co.Close()
	un := httptest.NewServer(NewHTTPHandler(upstream, true, false, s2compress, s2encrypt, key, ""))
	defer un.Close()

	// Initialize HTTP chunks stores, one RW and the other RO. Also make one that's
	// trying to get compressed data from a HTTP store that serves only uncompressed.
	coStoreURL, _ := url.Parse(co.URL)
	coStore, err := NewRemoteHTTPStore(coStoreURL, StoreOptions{Uncompressed: s1compress, Encrypted: s1encrypt, EncryptionKey: []byte("12345678901234567890123456789012")})
	if err != nil {
		t.Fatal(err)
	}
	unStoreURL, _ := url.Parse(un.URL)
	unStore, err := NewRemoteHTTPStore(unStoreURL, StoreOptions{Uncompressed: s2compress, Encrypted: s2encrypt, EncryptionKey: []byte("12345678901234567890123456789012")})
	if err != nil {
		t.Fatal(err)
	}
	invalidStore, err := NewRemoteHTTPStore(unStoreURL, StoreOptions{EncryptionKey: []byte("12345678901234567890123456789012")})
	if err != nil {
		t.Fatal(err)
	}

	// Make up some data and store it in the RW store
	dataIn := []byte("some data")
	chunkIn := NewChunkFromUncompressed(dataIn)
	id := chunkIn.ID(nil)

	// Try to get compressed chunks from a store that only serves uncompressed chunks
	if _, err := invalidStore.GetChunk(id); err == nil {
		t.Fatal("expected failure trying to get compressed chunks from uncompressed http store")
	}

	if err := coStore.StoreChunk(chunkIn); err != nil {
		t.Fatal(err)
	}

	// Check it's in the store when looking for compressed chunks
	coExists, err := coStore.HasChunk(id)
	if err != nil {
		t.Fatal(err)
	}
	if !coExists {
		t.Fatal("chunk not found in store")
	}

	// It's also visible when looking for uncompressed data
	unExists, err := unStore.HasChunk(id)
	if err != nil {
		t.Fatal(err)
	}
	if !unExists {
		t.Fatal("chunk not found in store")
	}

	// Send it uncompressed
	if err := unStore.StoreChunk(chunkIn); err != nil {
		t.Fatal(err)
	}

	// Try to get the uncompressed chunk
	if _, err := unStore.GetChunk(id); err != nil {
		t.Fatal(err)
	}
}
