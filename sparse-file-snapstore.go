package desync

import (
	"fmt"
	"os"
	"sync"

	"github.com/pkg/errors"
)

var _ Store = &SnapStore{}

// SnapStore
type SnapStore struct {
	dir       string
	name      string
	record    bool
	origin    Store
	w         WriteStore
	snapIndex Index
	muw       sync.RWMutex
}

// NewSnapStore creates an instance of a snapshot write store
func NewSnapStore(dir string, name string, opt StoreOptions, origin Store, originIndex Index, record bool) (*SnapStore, error) {
	snap := &SnapStore{
		dir:    dir,
		name:   name,
		record: record,
		origin: origin,
	}
	if dir != "" {
		w, err := NewLocalStore(dir, opt)
		if err != nil {
			fmt.Println("NewSnapStore err ", err)
			return &SnapStore{}, err
		}
		snap.w = w
	} else {
		snap.record = false
	}
	if name != "" {
		err := snap.loadSnapshot(name)
		if err != nil {
			fmt.Printf("NewSnapStore loadSnapshot err: %s\n", err)
			return snap, err
		}
		fmt.Printf("NewSnapStore loadSnapshot len(snap.snapIndex.Chunks) %d\n", len(snap.snapIndex.Chunks))
	} else {
		snap.snapIndex = Index{
			Index:  originIndex.Index,
			Chunks: originIndex.Chunks[:],
		}
	}
	if singleMon != nil { // log to monitor
		singleMon.SetSnapStore(snap)
	}
	return snap, nil
}

func (s *SnapStore) GetOffSize(i int) (int64, int64) {
	return int64(s.snapIndex.Chunks[i].Start), int64(s.snapIndex.Chunks[i].Size)
}

func (s *SnapStore) GetChunkID(i int) ChunkID {
	return s.snapIndex.Chunks[i].ID
}

func (s *SnapStore) GetIndexedChunk(i int, id ChunkID) (*Chunk, error) {
	// fmt.Printf("GetIndexedChunk s.name %s\n", s.name)
	// fmt.Printf("GetIndexedChunk s.snapIndex.Chunks %d\n", len(s.snapIndex.Chunks))
	c := s.snapIndex.Chunks[i]
	if c.ID == id {
		return s.origin.GetChunk(c.ID)
	}
	// fmt.Printf("GetIndexedChunk from write store %d\n", i)
	if s.w != nil {
		return s.w.GetChunk(c.ID)
	} else {
		return nil, errors.New("Snapstore writer not set. Can't reach that error !")
	}
}

// GetChunk reads and returns one (compressed!) chunk from the store
func (s *SnapStore) GetChunk(id ChunkID) (*Chunk, error) {
	if s.w != nil {
		return s.w.GetChunk(id)
	} else {
		return nil, errors.New("Snapstore writer not set. Can't reach that error !")
	}
}

// StoreChunk adds a new chunk to the store
func (s *SnapStore) StoreIndexedChunk(i int, chunk *Chunk) error {
	if s.w == nil {
		return errors.New("Snapstore writer not set. Can't reach that error !")
	}
	s.muw.Lock()
	defer s.muw.Unlock()
	// fmt.Printf("StoreIndexedChunk to write store\t %d\tID %s\n", i, chunk.ID().String())
	err := s.w.StoreChunk(chunk)
	if err == nil {
		s.snapIndex.Chunks[i].ID = chunk.ID()
	}
	return err
}

func (s *SnapStore) loadSnapshot(name string) error {
	if s.w == nil {
		return errors.New("loadSnapshot: Snapstore writer not set. Can't reach that error !")
	}
	s.muw.Lock()
	defer s.muw.Unlock()
	// Load named index
	s.name = name
	f, err := os.Open(name + ".snap.caibx")
	if err != nil {
		return err
	}
	defer f.Close()
	i, err := IndexFromReader(f)
	fmt.Printf("loadSnapshot len(i.Chunks) %d\n", len(i.Chunks))
	s.snapIndex = i
	return err
}

// takeSnapshot take a snapshot with the given name and set record for continuing
func (s *SnapStore) takeSnapshot(name string, record bool) error {
	if s.w == nil {
		return errors.New("takeSnapshot: Snapstore writer not set. Can't reach that error !")
	}
	if !s.record {
		return errors.New("takeSnapshot: Snapstore record not set.")
	}
	s.muw.Lock()
	defer s.muw.Unlock()
	// Write current index
	s.name = name
	s.record = record
	f, err := os.Create(name + ".snap.caibx")
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = s.snapIndex.WriteTo(f)
	return err
}

// HasChunk returns true if the chunk is in the store
func (s *SnapStore) HasChunk(id ChunkID) (bool, error) {
	if s.w != nil {
		return s.w.HasChunk(id)
	} else {
		return false, errors.New("Snapstore writer not set.Can't reach that error !")
	}
}

func (s *SnapStore) String() string {
	return s.dir
}

// Close the store.
func (s *SnapStore) Close() error {
	if s.w != nil {
		return s.w.Close()
	}
	return nil
}
