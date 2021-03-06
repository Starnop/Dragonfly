package progress

import (
	"github.com/dragonflyoss/Dragonfly/pkg/atomiccount"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"

	"github.com/willf/bitset"
)

type superState struct {
	// pieceBitSet maintains the piece bitSet of CID
	// which means that the status of each pieces of the task corresponding to taskID on the supernode.
	pieceBitSet *bitset.BitSet
}

type clientState struct {
	// pieceBitSet maintains the piece bitSet of CID
	// which means that the status of each pieces of the task on the peer corresponding to cid.
	pieceBitSet *bitset.BitSet

	// runningPiece maintains the pieces currently being downloaded from dstCID to srcCID.
	// key:pieceNum,value:dstPID
	runningPiece *syncmap.SyncMap
}

type peerState struct {
	// loadNum is the load of download services provided by the current node.
	//
	// This filed should be initialized in advance. If not, it will return an error.
	producerLoad *atomiccount.AtomicInt

	// clientErrorCount maintains the number of times that PeerID failed to downloaded from the other peer nodes.
	//
	// When this field is used, it will be initialized automatically with new AtomicInteger(0)
	// if it is not initialized.
	clientErrorCount *atomiccount.AtomicInt

	// serviceErrorCount maintains the number of times that the other peer nodes failed to downloaded from the PeerID.
	//
	// When this field is used, it will be initialized automatically with new AtomicInteger(0)
	// if it is not initialized.
	serviceErrorCount *atomiccount.AtomicInt

	// serviceDownTime the down time of the peer service.
	serviceDownTime int64
}

func newSuperState() *superState {
	return &superState{
		pieceBitSet: &bitset.BitSet{},
	}
}

func newClientState() *clientState {
	return &clientState{
		pieceBitSet:  &bitset.BitSet{},
		runningPiece: syncmap.NewSyncMap(),
	}
}

func newPeerState() *peerState {
	return &peerState{
		producerLoad:      atomiccount.NewAtomicInt(0),
		clientErrorCount:  atomiccount.NewAtomicInt(0),
		serviceErrorCount: atomiccount.NewAtomicInt(0),
	}
}
