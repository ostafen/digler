package table

const (
	// TableSize represents the fixed size of the internal hash table.
	// This value (2^16) is chosen to map directly to the uint16 hash space.
	TableSize = 65536 // Or 1 << 16 for clarity of its power-of-2 nature
)

// PrefixTable is a generic data structure that stores key-value pairs
// and allows for efficient prefix-based lookups and traversals.
// It uses a custom hashing mechanism to map byte prefixes to an internal
// 65536-byte (2^16) array, optimizing for small, byte-array keys.
// The `T` type parameter allows it to store any kind of value.
type PrefixTable[T any] struct {
	// table is a 65536-byte array used to mark the presence of prefixes.
	// Each byte in the array indicates the status of a hash collision point,
	// where the hash is derived from a key prefix.
	// It uses `none`, `presentMarker`, and `elemMarker` to differentiate
	// between no prefix, a prefix that is part of a longer key, and a prefix
	// that is itself a complete key associated with an element.
	table [TableSize]byte
	// elems stores the actual key-value pairs. Keys are stored as strings
	// for efficient map lookup, and values are of the generic type T.
	elems map[string]T
}

const (
	// none indicates that no key prefix hashes to this position in the table.
	none = iota
	// presentMarker indicates that a key prefix hashes to this position,
	// but it is only a prefix of a longer key, not a complete key itself.
	presentMarker
	// elemMarker indicates that a key prefix hashes to this position,
	// and this prefix is also a complete key stored in the `elems` map.
	elemMarker
)

// New creates and returns a new initialized PrefixTable.
// It allocates the internal map for storing elements.
func New[T any]() *PrefixTable[T] {
	return &PrefixTable[T]{
		elems: make(map[string]T),
	}
}

// Insert adds a new key-value pair to the PrefixTable.
//
// The `key` is a byte slice, and `v` is the value of generic type T.
//
// During insertion, it iterates through the `key` bytes, updating the
// internal `table` array. For each prefix of the key, it updates the
// corresponding hash entry in `table` to at least `presentMarker`.
// The entry corresponding to the full `key` is marked as `elemMarker`.
// Finally, the key-value pair is stored in the `elems` map.
//
// The hashing mechanism `h = (h << 2) + uint16(v)` effectively uses a
// 2-bit shift for each byte, allowing up to 8 bytes (16 bits / 2 bits_per_byte)
// of key information to directly influence the 16-bit hash, though
// longer keys will cause hash collisions at the 16-bit boundary.
// This is suitable for relatively short keys or scenarios where the
// first 8 bytes provide sufficient distinction for prefix matching.
func (t *PrefixTable[T]) Insert(key []byte, v T) {
	var h uint16 = 0
	for _, b := range key {
		h = (h << 2) + uint16(b)
		// `max` ensures that an `elemMarker` (if already set by a shorter key) isn't downgraded.
		t.table[h] = max(t.table[h], presentMarker)
	}
	// indicating that an element is associated with this full key.
	t.table[h] = elemMarker
	t.elems[string(key)] = v
}

// Get retrieves the value associated with a given `key` from the PrefixTable.
//
// It returns the value of type T and a boolean indicating whether the key
// was found (`true`) or not (`false`).
func (t *PrefixTable[T]) Get(key []byte) (T, bool) {
	v, found := t.elems[string(key)]
	return v, found
}

// Walk traverses the `PrefixTable` using a given `key` and executes the `onMatch`
// function for every complete key stored in the table that is a prefix of your `key`.
//
// For example, if your table contains "apple", "applet", and "apricot":
//   - If you call `Walk("appletie", onMatch)`, `onMatch` will be called twice:
//     first for "apple", and then for "applet".
//   - If you call `Walk("apricot", onMatch)`, `onMatch` will be called once for "apricot".
//   - If you call `Walk("application", onMatch)`, `onMatch` will not be called at all.
//
// The traversal stops as soon as a part of your `key` does not match any prefix
// in the table.
func (t *PrefixTable[T]) Walk(key []byte, onMatch func(T) bool) {
	var h uint16 = 0
	// Iterate through the key to find matching prefixes.
	for i, b := range key {
		h = (h << 2) + uint16(b)

		// Check the marker for the current prefix's hash.
		marker := t.table[h]
		if marker == none {
			// If `none`, no key starts with this prefix, so no longer prefixes can exist.
			return
		}

		if marker == elemMarker {
			// If `elemMarker`, this prefix itself is a complete key in the table.
			// Retrieve the value associated with this prefix and call the callback.
			// Note: string(key[:i+1]) creates a new string for the prefix.
			v, ok := t.elems[string(key[:i+1])]
			if ok && onMatch(v) {
				return
			}
		}
	}
}

// Size returns the number of unique key-value pairs currently stored in the table.
func (t *PrefixTable[T]) Size() int {
	return len(t.elems)
}
