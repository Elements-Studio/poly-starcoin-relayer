package db

import (
	"bytes"

	"golang.org/x/crypto/sha3"
)

const (
	right = 1
)

var (
	defaultValue = []byte{}
	hasher       = sha3.New256()
	leafPrefix   = []byte{0}
	nodePrefix   = []byte{1}
)

func UpdateRoot(path []byte, value []byte, sideNodes [][]byte, oldLeafData []byte) ([]byte, error) {
	var pathNode_0 []byte //pathNodes[0]
	if oldLeafData == nil {
		pathNode_0 = placeholder()
	} else {
		pathNode_0 = digest(oldLeafData)
	}
	valueHash := digest(value)
	currentHash, currentData := digestLeaf(path, valueHash)
	// if err := smt.nodes.Set(currentHash, currentData); err != nil {
	// 	return nil, err
	// }
	currentData = currentHash

	// If the leaf node that sibling nodes lead to has a different actual path
	// than the leaf node being updated, we need to create an intermediate node
	// with this leaf node and the new leaf node as children.
	//
	// First, get the number of bits that the paths of the two leaf nodes share
	// in common as a prefix.
	var commonPrefixCount int
	var oldValueHash []byte
	if bytes.Equal(pathNode_0, placeholder()) {
		commonPrefixCount = depth()
	} else {
		var actualPath []byte
		actualPath, oldValueHash = parseLeaf(oldLeafData)
		commonPrefixCount = countCommonPrefix(path, actualPath)
	}
	if commonPrefixCount != depth() {
		if getBitAtFromMSB(path, commonPrefixCount) == right {
			currentHash, currentData = digestNode(pathNode_0, currentData)
		} else {
			currentHash, currentData = digestNode(currentData, pathNode_0)
		}
		// err := smt.nodes.Set(currentHash, currentData)
		// if err != nil {
		// 	return nil, err
		// }
		currentData = currentHash
	} else if oldValueHash != nil {
		// // Short-circuit if the same value is being set
		// if bytes.Equal(oldValueHash, valueHash) {
		// 	return smt.root, nil
		// }
		// // If an old leaf exists, remove it
		// if err := smt.nodes.Delete(pathNodes[0]); err != nil {
		// 	return nil, err
		// }
		// if err := smt.values.Delete(path); err != nil {
		// 	return nil, err
		// }
	}

	// // All remaining path nodes are orphaned
	// for i := 1; i < len(pathNodes); i++ {
	// 	if err := smt.nodes.Delete(pathNodes[i]); err != nil {
	// 		return nil, err
	// 	}
	// }

	// The offset from the bottom of the tree to the start of the side nodes.
	// Note: i-offsetOfSideNodes is the index into sideNodes[]
	offsetOfSideNodes := depth() - len(sideNodes)

	for i := 0; i < depth(); i++ {
		var sideNode []byte

		if i-offsetOfSideNodes < 0 || sideNodes[i-offsetOfSideNodes] == nil {
			if commonPrefixCount != depth() && commonPrefixCount > depth()-1-i {
				// If there are no sidenodes at this height, but the number of
				// bits that the paths of the two leaf nodes share in common is
				// greater than this depth, then we need to build up the tree
				// to this depth with placeholder values at siblings.
				sideNode = placeholder()
			} else {
				continue
			}
		} else {
			sideNode = sideNodes[i-offsetOfSideNodes]
		}

		if getBitAtFromMSB(path, depth()-1-i) == right {
			currentHash, currentData = digestNode(sideNode, currentData)
		} else {
			currentHash, currentData = digestNode(currentData, sideNode)
		}
		// err := smt.nodes.Set(currentHash, currentData)
		// if err != nil {
		// 	return nil, err
		// }
		currentData = currentHash
	}
	// if err := smt.values.Set(path, value); err != nil {
	// 	return nil, err
	// }

	return currentHash, nil
}

func pathSize() int {
	return hasher.Size()
}

func depth() int {
	return pathSize() * 8
}

func placeholder() []byte {
	var zeroValue []byte = make([]byte, hasher.Size())
	return zeroValue
}

func digest(data []byte) []byte {
	hasher.Write(data)
	sum := hasher.Sum(nil)
	hasher.Reset()
	return sum
}

func digestLeaf(path []byte, leafData []byte) ([]byte, []byte) {
	value := make([]byte, 0, len(leafPrefix)+len(path)+len(leafData))
	value = append(value, leafPrefix...)
	value = append(value, path...)
	value = append(value, leafData...)

	hasher.Write(value)
	sum := hasher.Sum(nil)
	hasher.Reset()

	return sum, value
}

func digestNode(leftData []byte, rightData []byte) ([]byte, []byte) {
	value := make([]byte, 0, len(nodePrefix)+len(leftData)+len(rightData))
	value = append(value, nodePrefix...)
	value = append(value, leftData...)
	value = append(value, rightData...)

	hasher.Write(value)
	sum := hasher.Sum(nil)
	hasher.Reset()

	return sum, value
}

func parseLeaf(data []byte) ([]byte, []byte) {
	return data[len(leafPrefix) : pathSize()+len(leafPrefix)], data[len(leafPrefix)+pathSize():]
}

func countCommonPrefix(data1 []byte, data2 []byte) int {
	count := 0
	for i := 0; i < len(data1)*8; i++ {
		if getBitAtFromMSB(data1, i) == getBitAtFromMSB(data2, i) {
			count++
		} else {
			break
		}
	}
	return count
}

// getBitAtFromMSB gets the bit at an offset from the most significant bit
func getBitAtFromMSB(data []byte, position int) int {
	if int(data[position/8])&(1<<(8-1-uint(position)%8)) > 0 {
		return 1
	}
	return 0
}
