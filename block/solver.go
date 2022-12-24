package block

type Solver interface {
	Solve(*Block) error
	Verify(Block) error
}
