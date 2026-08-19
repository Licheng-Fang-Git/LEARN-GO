package btree

type Node struct{
	keys [][]byte
	vals [][]byte
	kids []*Node
}
