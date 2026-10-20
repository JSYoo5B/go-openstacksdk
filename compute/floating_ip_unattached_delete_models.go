package compute

// Each eligible inventory row retains its independently owned view and the
// deletion result, including accepted attempts when a later error stops work.
type UnattachedFloatingIPDelete struct {
	FloatingIP *FloatingIPRecord
	ID         string
	Deletion   *DeleteFloatingIPResult
	Error      error
}

// Count counts completed Deleted=true results, including DOWN-policy success;
// it does not count accepted requests as proof of deletion or actual absence.
// On success, AllDeleted maps Python's integer Count versus boolean false.
// An empty or ineligible operation returns Count=0 and AllDeleted=true.
// On error, Inventory/Items/Count are partial evidence and AllDeleted is false.
type DeleteUnattachedFloatingIPsResult struct {
	Eligible   bool
	Inventory  *FloatingIPQueryResult
	Items      []*UnattachedFloatingIPDelete
	Count      int
	AllDeleted bool
}
