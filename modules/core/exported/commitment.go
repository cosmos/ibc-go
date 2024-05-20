package exported

// ICS 023 Types Implementation
//
// This file includes types defined under
// https://github.com/cosmos/ibc/tree/master/spec/core/ics-023-vector-commitments

// spec:Path and spec:Value are defined as bytestring

// Root implements spec:CommitmentRoot.
// A root is constructed from a set of key-value pairs,
// and the inclusion or non-inclusion of an arbitrary key-value pair
// can be proven with the proof.
type Root interface {
	GetHash() []byte
	Empty() bool
}

// Prefix implements spec:CommitmentPrefix.
// Prefix represents the common "prefix" that a set of keys shares.
type Prefix interface {
	Bytes() []byte
	Empty() bool
}

// Path implements spec:CommitmentPath.
// A path is the additional information provided to the verification function.
type Path interface {
	Empty() bool
}
