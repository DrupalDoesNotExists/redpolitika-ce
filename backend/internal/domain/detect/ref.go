package detect

// RefNode delegates detection to another rule's detect tree.
// RefNode is created during phase 1 parsing with just a refID,
// then Resolve() is called during phase 2 with the actual Node.
type RefNode struct {
	refID string
	node  Node // resolved later
}

// NewRefNode creates a RefNode with the given rule ID reference.
func NewRefNode(refID string) *RefNode {
	return &RefNode{refID: refID}
}

// Detect delegates to the resolved node. Returns empty set if not yet resolved.
func (n *RefNode) Detect(text string) []MatchRange {
	if n.node == nil {
		return nil
	}
	return n.node.Detect(text)
}

// Resolve wires the referenced rule's detect node.
func (n *RefNode) Resolve(node Node) { n.node = node }

// RefID returns the referenced rule ID string.
func (n *RefNode) RefID() string { return n.refID }

// IsResolved reports whether this ref has been resolved.
func (n *RefNode) IsResolved() bool { return n.node != nil }
