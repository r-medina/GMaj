package gmaj

import (
	"errors"

	"github.com/r-medina/gmaj/gmajpb"

	"golang.org/x/net/context"
)

var (
	emptyRemote = &gmajpb.Node{}
	mt          = &gmajpb.MT{}
)

// GetPredecessor gets the predecessor on the node.
func (node *Node) GetPredecessor(context.Context, *gmajpb.MT) (*gmajpb.Node, error) {
	node.predMtx.RLock()
	pred := node.Predecessor
	node.predMtx.RUnlock()

	if pred == nil {
		return emptyRemote, nil
	}

	return pred, nil
}

// GetSuccessor gets the successor on the node..
func (node *Node) GetSuccessor(context.Context, *gmajpb.MT) (*gmajpb.Node, error) {
	node.succMtx.RLock()
	succ := node.Successor
	node.succMtx.RUnlock()

	if succ == nil {
		return emptyRemote, nil
	}

	return succ, nil
}

// SetPredecessor sets the predecessor on the node.
func (node *Node) SetPredecessor(
	ctx context.Context, pred *gmajpb.Node,
) (*gmajpb.MT, error) {
	node.predMtx.Lock()
	node.Predecessor = pred
	node.predMtx.Unlock()

	return mt, nil
}

// SetSuccessor sets the successor on the node.
func (node *Node) SetSuccessor(
	ctx context.Context, succ *gmajpb.Node,
) (*gmajpb.MT, error) {
	node.succMtx.Lock()
	node.Successor = succ
	node.succMtx.Unlock()

	return mt, nil
}

// Notify is called when remoteNode thinks it's our predecessor.
func (node *Node) Notify(
	ctx context.Context, remoteNode *gmajpb.Node,
) (*gmajpb.MT, error) {
	node.notify(remoteNode)

	// If node.Predecessor is nil at this point, we were trying to notify
	// ourselves. Otherwise, to succeed, we must check that the successor
	// was correctly updated.
	node.predMtx.Lock()
	defer node.predMtx.Unlock()
	if node.Predecessor != nil && !idsEqual(node.Predecessor.Id, remoteNode.Id) {
		return mt, errors.New("remoteNode is not node's predecessor")
	}

	return mt, nil
}

// ClosestPrecedingFinger will find the closest preceding entry in the finger
// table based on the id.
func (node *Node) ClosestPrecedingFinger(
	ctx context.Context, id *gmajpb.ID,
) (*gmajpb.Node, error) {
	remoteNode := node.closestPrecedingFinger(id.Id)
	if remoteNode == nil {
		return emptyRemote, errors.New("MT node closest preceding finger")
	}

	return remoteNode, nil
}

// FindSuccessor finds the successor, error if nil.
func (node *Node) FindSuccessor(
	ctx context.Context, id *gmajpb.ID,
) (*gmajpb.Node, error) {
	succ, err := node.findSuccessor(id.Id)
	if err != nil {
		return emptyRemote, err
	}

	if succ == nil {
		return emptyRemote, errors.New("cannot find successor")
	}

	return succ, nil
}

// Get returns the value of the key requested at the node.
func (node *Node) Get(ctx context.Context, key *gmajpb.Key) (*gmajpb.Val, error) {
	val, err := node.get(key)
	if err != nil {
		return nil, err
	}

	return &gmajpb.Val{Val: val}, nil
}

// Put stores a key value pair on the node.
func (node *Node) Put(ctx context.Context, keyVal *gmajpb.KeyVal) (*gmajpb.MT, error) {
	if err := node.put(keyVal); err != nil {
		return nil, err
	}

	return mt, nil
}

// TransferKeys transfers the appropriate keys on this node
// to the remote node specified in the request.
func (node *Node) TransferKeys(
	ctx context.Context, tmsg *gmajpb.TransferKeysReq,
) (*gmajpb.MT, error) {
	if err := node.transferKeys(tmsg); err != nil {
		return nil, err
	}

	return mt, nil
}
