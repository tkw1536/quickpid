//spellchecker:words idgen
package idgen_test

//spellchecker:words errors iter testing github quickpid internal idgen
import (
	"errors"
	"fmt"
	"io"
	"iter"
	"testing"

	"github.com/tkw1536/quickpid/internal/idgen"
)

func ExamplePullIDs() {
	next, stop := idgen.PullIDs(func(round uint64) iter.Seq2[string, error] {
		switch round {
		case 0:
			return seq("a", "b")
		case 1:
			return seq("c")
		default:
			return seq()
		}
	})
	defer stop()

	for {
		id, err := next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			fmt.Println("err:", err)
			continue
		}
		fmt.Println(id)
	}

	// Output:
	// a
	// b
	// c
}

var errBoom = errors.New("boom")

func ExamplePullIDs_errorAbandonsSequence() {
	next, stop := idgen.PullIDs(func(round uint64) iter.Seq2[string, error] {
		switch round {
		case 0:
			return func(yield func(string, error) bool) {
				if !yield("keep", nil) {
					return
				}
				if !yield("skip-rest", errBoom) {
					return
				}
				yield("not-reached", nil)
			}
		case 1:
			return seq("after-error")
		default:
			return seq()
		}
	})
	defer stop()

	for range 4 {
		id, err := next()
		if err != nil {
			fmt.Printf("%q %v\n", id, err)
			continue
		}
		fmt.Printf("%q\n", id)
	}

	// Output:
	// "keep"
	// "skip-rest" boom
	// "after-error"
	// "" EOF
}

func TestPullIDs_emptyFirstSequence(t *testing.T) {
	t.Parallel()

	next, stop := idgen.PullIDs(func(uint64) iter.Seq2[string, error] {
		return seq()
	})
	defer stop()

	id, err := next()
	if id != "" || !errors.Is(err, io.EOF) {
		t.Fatalf("next() = %q, %v; want \"\", io.EOF", id, err)
	}
}

func TestPullIDs_eofAfterStop(t *testing.T) {
	t.Parallel()

	next, stop := idgen.PullIDs(func(round uint64) iter.Seq2[string, error] {
		if round == 0 {
			return seq("only")
		}
		return seq()
	})

	id, err := next()
	if id != "only" || err != nil {
		t.Fatalf("next() = %q, %v; want \"only\", nil", id, err)
	}

	stop()

	id, err = next()
	if id != "" || !errors.Is(err, io.EOF) {
		t.Fatalf("next() after stop = %q, %v; want \"\", io.EOF", id, err)
	}
}

func TestPullIDs_rounds(t *testing.T) {
	t.Parallel()

	var calls []uint64
	next, stop := idgen.PullIDs(func(round uint64) iter.Seq2[string, error] {
		calls = append(calls, round)
		switch round {
		case 0:
			return seq("x")
		case 1:
			return seq("y", "z")
		default:
			return seq()
		}
	})
	defer stop()

	want := []string{"x", "y", "z"}
	for i, wantID := range want {
		id, err := next()
		if err != nil || id != wantID {
			t.Fatalf("next() #%d = %q, %v; want %q, nil", i, id, err, wantID)
		}
	}

	id, err := next()
	if id != "" || !errors.Is(err, io.EOF) {
		t.Fatalf("next() after end = %q, %v; want \"\", io.EOF", id, err)
	}

	wantCalls := []uint64{0, 1, 2}
	if len(calls) != len(wantCalls) {
		t.Fatalf("generator calls = %v; want %v", calls, wantCalls)
	}
	for i := range wantCalls {
		if calls[i] != wantCalls[i] {
			t.Fatalf("generator calls = %v; want %v", calls, wantCalls)
		}
	}
}

func seq(values ...string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for _, v := range values {
			if !yield(v, nil) {
				return
			}
		}
	}
}
