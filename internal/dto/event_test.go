package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeAttributesKeepsAnIntegerPastTwoToTheFiftyThree(t *testing.T) {
	var event Event

	require.NoError(t, event.DecodeAttributes(`{"offset":9007199254740993}`))

	encoded, err := json.Marshal(event.Attributes)
	require.NoError(t, err)
	require.Equal(t, `{"offset":9007199254740993}`, string(encoded))
}

func TestDecodeAttributesNamesTheEventItFailsOn(t *testing.T) {
	event := Event{Kind: "skill", Session: "s1", Timestamp: "2026-10-15T10:00:00Z"}

	err := event.DecodeAttributes(`{"broken"`)

	require.ErrorContains(t, err, "attributes of the skill event in session s1 at 2026-10-15T10:00:00Z: ")
}
