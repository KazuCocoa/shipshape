// Binary entrystream provides tools to manipulate a stream of delimited Entry
// messages. By default, entrystream does nothing to the entry stream.
//
// Examples:
//   $ ... | entrystream                      # Passes through proto entry stream unchanged
//   $ ... | entrystream --sort               # Sorts the entry stream into GraphStore order
//   $ ... | entrystream --write_json         # Prints entry stream as JSON
//   $ ... | entrystream --write_json --sort  # Sorts the JSON entry stream into GraphStore order
//   $ ... | entrystream --entrysets          # Prints combined entry sets as JSON
//   $ ... | entrystream --count              # Prints the number of entries in the incoming stream
//   $ ... | entrystream --read_json          # Reads entry stream as JSON and prints a proto stream
package main

import (
	"container/heap"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"third_party/kythe/go/platform/delimited"
	"third_party/kythe/go/storage"
	"third_party/kythe/go/storage/stream"

	spb "third_party/kythe/proto/storage_proto"

	"code.google.com/p/goprotobuf/proto"
)

type entrySet struct {
	Source   *spb.VName `json:"source"`
	Target   *spb.VName `json:"target,omitempty"`
	EdgeKind string     `json:"edge_kind,omitempty"`

	Properties map[string]string `json:"properties"`
}

var (
	readJSON   = flag.Bool("read_json", false, "Assume stdin is a stream of JSON entries instead of protobufs")
	writeJSON  = flag.Bool("write_json", false, "Print JSON stream as output")
	sortStream = flag.Bool("sort", false, "Sort entry stream into GraphStore order")
	entrySets  = flag.Bool("entrysets", false, "Print Entry protos as JSON EntrySets (implies --sort and --write_json)")
	countOnly  = flag.Bool("count", false, "Only print the count of protos streamed")
)

func main() {
	flag.Parse()

	in := os.Stdin
	var entries <-chan *spb.Entry
	if *readJSON {
		entries = stream.ReadJSONEntries(in)
	} else {
		entries = stream.ReadEntries(in)
	}
	if *sortStream || *entrySets {
		unsortedEntries := entries
		ch := make(chan *spb.Entry)
		entries = ch
		sorted := sortedEntries(make([]*spb.Entry, 0))
		heap.Init(&sorted)
		go func() {
			for entry := range unsortedEntries {
				heap.Push(&sorted, entry)
			}
			for len(sorted) > 0 {
				ch <- heap.Pop(&sorted).(*spb.Entry)
			}
			close(ch)
		}()
	}

	encoder := json.NewEncoder(os.Stdout)
	wr := delimited.NewWriter(os.Stdout)

	var set entrySet
	entryCount := 0
	for entry := range entries {
		if *countOnly {
			entryCount++
		} else if *entrySets {
			if !proto.Equal(set.Source, entry.Source) || !proto.Equal(set.Target, entry.Target) || set.EdgeKind != entry.GetEdgeKind() {
				if len(set.Properties) != 0 {
					failOnErr(encoder.Encode(set))
				}
				set.Source = entry.Source
				set.EdgeKind = entry.GetEdgeKind()
				set.Target = entry.Target
				set.Properties = make(map[string]string)
			}
			set.Properties[entry.GetFactName()] = string(entry.GetFactValue())
		} else if *writeJSON {
			failOnErr(encoder.Encode(entry))
		} else {
			rec, err := proto.Marshal(entry)
			failOnErr(err)
			failOnErr(wr.Put(rec))
		}
	}
	if len(set.Properties) != 0 {
		failOnErr(encoder.Encode(set))
	}
	if *countOnly {
		fmt.Println(entryCount)
	}
}

func failOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type sortedEntries []*spb.Entry

func (m sortedEntries) Len() int { return len(m) }
func (m sortedEntries) Less(i, j int) bool {
	return storage.EntryLess(m[i], m[j])
}
func (m sortedEntries) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}
func (m *sortedEntries) Push(x interface{}) {
	*m = append(*m, x.(*spb.Entry))
}
func (m *sortedEntries) Pop() interface{} {
	old := *m
	n := len(old)
	item := old[n-1]
	*m = old[0 : n-1]
	return item
}
