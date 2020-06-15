package flatend

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"testing/quick"
)

func TestServiceRequestPacket(t *testing.T) {
	var dst []byte
	f := func(expected ServiceRequestPacket) bool {
		actual, err := UnmarshalServiceRequestPacket(expected.AppendTo(dst[:0]))
		return assert.NoError(t, err) && assert.EqualValues(t, expected, actual)
	}
	require.NoError(t, quick.Check(f, nil))
}

func TestServiceResponsePacket(t *testing.T) {
	var dst []byte
	f := func(expected ServiceResponsePacket) bool {
		actual, err := UnmarshalServiceResponsePacket(expected.AppendTo(dst[:0]))
		return assert.NoError(t, err) && assert.EqualValues(t, expected, actual)
	}
	require.NoError(t, quick.Check(f, nil))
}

func TestDataPacket(t *testing.T) {
	var dst []byte
	f := func(expected DataPacket) bool {
		actual, err := UnmarshalDataPacket(expected.AppendTo(dst[:0]))
		return assert.NoError(t, err) && assert.EqualValues(t, expected, actual)
	}
	require.NoError(t, quick.Check(f, nil))
}
