package model

// FrontendBundleSlot names valid UI injection points (A17).
type FrontendBundleSlot string

const (
	FrontendBundleSlotRoot   FrontendBundleSlot = "root"
	FrontendBundleSlotNavbar FrontendBundleSlot = "navbar"
	FrontendBundleSlotFooter FrontendBundleSlot = "footer"
	FrontendBundleSlotEditor FrontendBundleSlot = "editor"
	FrontendBundleSlotAside  FrontendBundleSlot = "aside"
)

var validFrontendBundleSlots = map[FrontendBundleSlot]struct{}{
	FrontendBundleSlotRoot:   {},
	FrontendBundleSlotNavbar: {},
	FrontendBundleSlotFooter: {},
	FrontendBundleSlotEditor: {},
	FrontendBundleSlotAside:  {},
}

// FrontendBundle — plugin-provided JS/CSS bundle for a UI slot (A17).
type FrontendBundle struct {
	id     string
	plugin string
	slot   FrontendBundleSlot
	entry  string
	css    []string
}

// NewFrontendBundle creates a validated FrontendBundle.
// ID, Plugin, Slot, and Entry are required; Slot must be one of the valid slots.
func NewFrontendBundle(id, plugin, slot, entry string, css []string) (FrontendBundle, error) {
	if id == "" {
		return FrontendBundle{}, &DomainError{Op: "NewFrontendBundle", Message: "id must not be empty"}
	}
	if plugin == "" {
		return FrontendBundle{}, &DomainError{Op: "NewFrontendBundle", Message: "plugin must not be empty"}
	}
	if slot == "" {
		return FrontendBundle{}, &DomainError{Op: "NewFrontendBundle", Message: "slot must not be empty"}
	}
	if entry == "" {
		return FrontendBundle{}, &DomainError{Op: "NewFrontendBundle", Message: "entry must not be empty"}
	}

	s := FrontendBundleSlot(slot)
	if _, ok := validFrontendBundleSlots[s]; !ok {
		return FrontendBundle{}, &DomainError{
			Op:      "NewFrontendBundle",
			Message: "invalid slot: " + slot,
		}
	}

	return FrontendBundle{
		id:     id,
		plugin: plugin,
		slot:   s,
		entry:  entry,
		css:    cloneStringSlice(css),
	}, nil
}

func (b FrontendBundle) ID() string                  { return b.id }
func (b FrontendBundle) Plugin() string              { return b.plugin }
func (b FrontendBundle) Slot() FrontendBundleSlot    { return b.slot }
func (b FrontendBundle) Entry() string               { return b.entry }
func (b FrontendBundle) CSS() []string               { return cloneStringSlice(b.css) }

func cloneStringSlice(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}
