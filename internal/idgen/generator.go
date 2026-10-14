//spellchecker:words idgen
package idgen

//spellchecker:words iter
import (
	"io"
	"iter"
)

// An IDGenerationFunction produces candidate identifiers as a sequence of (value, error) pairs.
//
// The round argument starts at 0 and increases each time the previous sequence has been fully
// consumed (or abandoned after a non-nil error). Concurrent ID generation operations each keep
// an independent round counter; rounds are not shared across operations.
//
// The runtime holds several such functions as methods with extra parameters.
// Use [PullIDs] to adapt an [IDGenerationFunction] into a pull-based API.
type IDGenerationFunction = func(round uint64) iter.Seq2[string, error]

// PullIDs starts one ID generation operation for the given [IDGenerationFunction].
// It returns a pull-based API: next yields the next candidate, and stop ends the operation
// early. Within the operation, consecutive sequences from generator(0), generator(1), ... are
// joined into one stream and share a single round counter.
//
//   - An empty sequence ends the operation; later next calls return [io.EOF].
//   - A non-nil yielded error abandons that invocation's remaining values; generator is then
//     called again with a higher round.
//   - After the operation has ended (empty sequence or stop), next returns [io.EOF].
//
// The caller must invoke stop when no further next calls will be made.
// Call PullIDs once per operation; concurrent operations must each use their own PullIDs call.
func PullIDs(generator IDGenerationFunction) (next func() (string, error), stop func()) {
	pull, stop := iter.Pull2(func(yield func(string, error) bool) {
		var round uint64
		for {
			sequenceWasEmpty := true
			for pid, err := range generator(round) {
				sequenceWasEmpty = false
				if !yield(pid, err) {
					return
				}
				if err != nil {
					break
				}
			}
			if sequenceWasEmpty {
				return
			}
			round++
		}
	})

	return func() (string, error) {
		pid, err, ok := pull()
		if !ok {
			return "", io.EOF
		}
		return pid, err
	}, stop
}
