package main

import "crypto/sha256"

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

type MerkleTree struct {
	root *MerkleNode
}

func NewMerkleNode(left *MerkleNode, right *MerkleNode, data []byte) *MerkleNode {
	node := new(MerkleNode)
	if left == nil && right == nil {
		dataHash := sha256.Sum256(data)
		node.Data = dataHash[:]
	} else if left != nil && right != nil {
		prevHash := append(left.Data, right.Data...)
		dataHash := sha256.Sum256(prevHash)
		node.Data = dataHash[:]
	}
	node.Left = left
	node.Right = right
	return node
}

func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode

	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	for _, tmp := range data {
		node := NewMerkleNode(nil, nil, tmp)
		nodes = append(nodes, *node)
	}

	for i := 0; i < len(data)/2; i++ {
		var newLevel []MerkleNode

		for j := 0; j < len(nodes); j += 2 {
			node := *NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			newLevel = append(newLevel, node)
		}

		nodes = newLevel
	}

	tree := &MerkleTree{&nodes[0]}
	return tree
}
