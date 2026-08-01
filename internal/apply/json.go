package apply

import "encoding/json"

// MarshalJSON preserves contract metadata for callers that construct Output
// values directly, including the CLI aggregation path.
func (output Output) MarshalJSON() ([]byte, error) {
	if output.Kind == "" {
		output.Kind = OutputKind
	}
	if output.Version == 0 {
		output.Version = OutputVersion
	}
	type outputAlias Output
	return json.Marshal(outputAlias(output))
}
