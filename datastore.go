//
//  contains the API and interal functions to interact with the Key-Value store
//  that the Chord ring is providing
//

package gmaj

import (
	"errors"
	"fmt"
	"time"

	"github.com/r-medina/gmaj/gmajpb"
)

var errNoDatastore = errors.New("Node does not have a datastore")

//
// External API Into Datastore
//

// Get a value in the datastore, provided an abitrary node in the ring
func Get(node *Node, key string) (string, error) {
	if node == nil {
		return "", errors.New("Node cannot be nil")
	}

	remoteNode, err := node.locate(key)
	if err != nil {
		return "", err
	}

	// TODO(asubiotto): Smart retries on error. Implement channel that notifies
	// when stabilize has been called.

	// Retry on error because it might be due to temporary unavailability
	// (e.g. write happened while transferring nodes).
	value, err := node.GetRPC(remoteNode, key)
	if err != nil {
		<-time.After(cfg.RetryInterval)
		remoteNode, err = node.locate(key)
		if err != nil {
			return "", err
		}

		return node.GetRPC(remoteNode, key)
	}

	return value, nil
}

// Put a key/value in the datastore, provided an abitrary node in the ring.
// This is useful for testing.
func Put(node *Node, key string, value string) error {
	if node == nil {
		return errors.New("Node cannot be nil")
	}

	remoteNode, err := node.locate(key)
	if err != nil {
		return err
	}

	return node.PutRPC(remoteNode, key, value)
}

// locate helps find the appropriate node in the ring.
func (node *Node) locate(key string) (*gmajpb.RemoteNode, error) {
	return node.FindSuccessorRPC(&node.remoteNode, HashKey(key))
}

// obtainNewKeys is called when a node joins a ring and wants to request keys
// from its successor.
func (node *Node) obtainNewKeys() error {
	node.succMtx.RLock()
	defer node.succMtx.RUnlock()

	// TODO(asubiotto): Test the case where there are two nodes floating around
	// that need keys.
	// Assume new predecessor has been set.
	prevPredecessor, err := node.GetPredecessorRPC(node.Successor)
	if err != nil {
		return err
	}

	return node.TransferKeysRPC(
		node.Successor,
		node.remoteNode.Id,
		prevPredecessor,
	) // implicitly correct even when prevPredecessor.ID == nil
}

//
// RPCs to assist with interfacing with the datastore ring
//

func (node *Node) get(key *gmajpb.Key) (string, error) {

	if node.dataStore == nil {
		return "", errNoDatastore
	}

	node.dsMtx.RLock()
	val, ok := node.dataStore[key.Key]
	node.dsMtx.RUnlock()
	if !ok {
		return "", errors.New("Key does not exist")
	}

	return val, nil
}

func (node *Node) put(keyVal *gmajpb.KeyVal) error {
	if node.dataStore == nil {
		return errNoDatastore
	}

	key := keyVal.Key
	val := keyVal.Val

	node.dsMtx.RLock()
	_, exists := node.dataStore[key]
	node.dsMtx.RUnlock()
	if exists {
		return errors.New("Cannot modify an existing value")
	}

	node.dsMtx.Lock()
	node.dataStore[key] = val
	node.dsMtx.Unlock()

	return nil
}

func (node *Node) transferKeys(tmsg *gmajpb.TransferMsg) error {
	toNode := tmsg.ToNode
	if IDsEqual(toNode.Id, node.ID()) {
		return nil
	}

	node.dsMtx.Lock()
	defer node.dsMtx.Unlock()

	toDelete := []string{}
	for key, val := range node.dataStore {
		hashedKey := HashKey(key)

		// Check that the hashed_key lies in the correct range before putting
		// the value in our predecessor.
		if BetweenRightIncl(hashedKey, tmsg.FromID, toNode.Id) {
			if err := node.PutRPC(toNode, key, val); err != nil {
				return err
			}

			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {
		delete(node.dataStore, key)
	}

	return nil
}

// PrintDataStore write the contents of a node's data store to stdout.
func PrintDataStore(node *Node) {
	node.dsMtx.RLock()
	fmt.Printf(
		"Node-%v datastore: %v\n",
		IDToString(node.remoteNode.Id), node.dataStore,
	)
	node.dsMtx.RUnlock()
}
