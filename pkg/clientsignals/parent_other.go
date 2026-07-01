//go:build !linux && !darwin && !windows

package clientsignals

// lookupParentName has no implementation on unrecognized platforms; the
// caller classifies "" as the "other" bucket.
func lookupParentName(_ int) string {
	return ""
}
