// Package xrefs contains the xref.Service interface and a simple implementation
// backed by a storage.GraphStore
package xrefs

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sync"

	"third_party/kythe/go/storage"
	"third_party/kythe/go/util/kytheuri"
	"third_party/kythe/go/util/schema"
	"third_party/kythe/go/util/stringset"

	spb "third_party/kythe/proto/storage_proto"
	xpb "third_party/kythe/proto/xref_proto"

	"code.google.com/p/goprotobuf/proto"
)

// Service provides access to a Kythe graph for fast access to cross-references.
type Service interface {
	// Nodes returns a subset of the facts for each of the requested nodes.
	Nodes(*xpb.NodesRequest) (*xpb.NodesReply, error)

	// Edges returns a subset of the outbound edges for each of a set of requested
	// nodes.
	Edges(*xpb.EdgesRequest) (*xpb.EdgesReply, error)

	// Decorations returns an index of the nodes and edges associated with a
	// particular file node.
	Decorations(*xpb.DecorationsRequest) (*xpb.DecorationsReply, error)
}

// A GraphStoreService implements the Service interface for a GraphStore that
// has been preprocessed for cross-references.
// TODO(schroederc): parallelize GraphStore calls
type GraphStoreService struct {
	gs storage.GraphStore
}

// NewGraphStoreService returns a new GraphStoreService given an
// existing GraphStore.
func NewGraphStoreService(gs storage.GraphStore) *GraphStoreService {
	return &GraphStoreService{gs}
}

// Nodes implements part of the Service interface.
func (g *GraphStoreService) Nodes(req *xpb.NodesRequest) (*xpb.NodesReply, error) {
	patterns := convertFilters(req.Filter)

	var names []*spb.VName
	for _, ticket := range req.Ticket {
		name, err := kytheuri.ToVName(ticket)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	var nodes []*xpb.NodeInfo
	for i, vname := range names {
		info := &xpb.NodeInfo{Ticket: &req.Ticket[i]}
		entries := make(chan *spb.Entry)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for entry := range entries {
				if len(patterns) == 0 || matchesAny(entry.GetFactName(), patterns) {
					info.Fact = append(info.Fact, entryToFact(entry))
				}
			}
		}()
		err := g.gs.Read(&spb.ReadRequest{Source: vname}, entries)
		close(entries)
		if err != nil {
			return nil, err
		}
		wg.Wait()
		nodes = append(nodes, info)
	}
	return &xpb.NodesReply{Node: nodes}, nil
}

// Edges implements part of the Service interface.
// TODO(schroederc): handle pagination requests
func (g *GraphStoreService) Edges(req *xpb.EdgesRequest) (*xpb.EdgesReply, error) {
	if req.GetPageToken() != "" {
		return nil, errors.New("UNIMPLEMENTED: page_token")
	}

	patterns := convertFilters(req.Filter)
	allowedKinds := stringset.New(req.Kind...)
	targetSet := stringset.New()
	reply := new(xpb.EdgesReply)

	for _, ticket := range req.Ticket {
		vname, err := kytheuri.ToVName(ticket)
		if err != nil {
			return nil, fmt.Errorf("invalid ticket %q: %v", ticket, err)
		}

		var (
			// EdgeKind -> StringSet<TargetTicket>
			filteredEdges = make(map[string]stringset.Set)
			filteredFacts []*xpb.Fact
		)

		entries := make(chan *spb.Entry)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for entry := range entries {
				edgeKind := entry.GetEdgeKind()
				if edgeKind == "" {
					// node fact
					if len(patterns) > 0 && matchesAny(entry.GetFactName(), patterns) {
						filteredFacts = append(filteredFacts, entryToFact(entry))
					}
				} else {
					// edge
					if len(req.Kind) == 0 || allowedKinds.Contains(edgeKind) {
						targets := filteredEdges[edgeKind]
						if targets == nil {
							targets = stringset.New()
							filteredEdges[edgeKind] = targets
						}
						targets.Add(vnameToTicket(entry.Target))
					}
				}
			}
		}()
		err = g.gs.Read(&spb.ReadRequest{
			Source:   vname,
			EdgeKind: proto.String("*"),
		}, entries)
		close(entries)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve entries for ticket %q", ticket)
		}
		wg.Wait()

		// Only add a EdgeSet if there are targets for the requested edge kinds.
		if len(filteredEdges) > 0 {
			var groups []*xpb.EdgeSet_Group
			for edgeKind, targets := range filteredEdges {
				g := &xpb.EdgeSet_Group{Kind: proto.String(edgeKind)}
				for target := range targets {
					g.TargetTicket = append(g.TargetTicket, target)
					targetSet.Add(target)
				}
				groups = append(groups, g)
			}
			reply.EdgeSet = append(reply.EdgeSet, &xpb.EdgeSet{
				SourceTicket: proto.String(ticket),
				Group:        groups,
			})

			// In addition, only add a NodeInfo if the filters have resulting facts.
			if len(filteredFacts) > 0 {
				reply.Node = append(reply.Node, &xpb.NodeInfo{
					Ticket: proto.String(ticket),
					Fact:   filteredFacts,
				})
			}
		}
	}

	// Ensure reply.Node is a unique set by removing already requested nodes from targetSet
	for _, n := range reply.Node {
		targetSet.Remove(n.GetTicket())
	}

	// Batch request all leftover target nodes
	nodesReply, err := g.Nodes(&xpb.NodesRequest{Ticket: targetSet.Slice(), Filter: req.Filter})
	if err != nil {
		return nil, fmt.Errorf("failure getting target nodes: %v", err)
	}
	reply.Node = append(reply.Node, nodesReply.Node...)

	return reply, nil
}

// Decorations implements part of the Service interface.
// TODO(schroederc): implement location windowing
// TODO(schroederc): implement patching
func (g *GraphStoreService) Decorations(req *xpb.DecorationsRequest) (*xpb.DecorationsReply, error) {
	if len(req.DirtyBuffer) > 0 {
		return nil, errors.New("UNIMPLEMENTED: patching")
	} else if req.Location.GetKind() != xpb.Location_FILE {
		return nil, errors.New("UNIMPLEMENTED: span locations")
	}

	fileVName, err := kytheuri.ToVName(req.Location.GetTicket())
	if err != nil {
		return nil, fmt.Errorf("invalid file ticket %q: %v", req.Location.GetTicket(), err)
	}

	reply := &xpb.DecorationsReply{Location: req.Location}

	// Handle DecorationsRequest.SourceText switch
	if req.GetSourceText() {
		text, encoding, err := getSourceText(g.gs, fileVName)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve file text: %v", err)
		}
		reply.SourceText = text
		reply.Encoding = &encoding
	}

	// Handle DecorationsRequest.References switch
	if req.GetReferences() {
		// Traverse the following chain of edges:
		//   file --%/kythe/edge/childof-> []anchor --forwardEdgeKind-> []target
		//
		// Add []anchor and []target nodes to reply.Node
		// Add all {anchor, forwardEdgeKind, target} tuples to reply.Reference

		children, err := getEdges(g.gs, fileVName, func(e *spb.Entry) bool {
			return e.GetEdgeKind() == revChildOfEdgeKind
		})
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve file children: %v", err)
		}

		targetSet := stringset.New()
		for _, edge := range children {
			anchor := edge.Target
			anchorNodeReply, err := g.Nodes(&xpb.NodesRequest{Ticket: []string{vnameToTicket(anchor)}})
			if err != nil {
				return nil, fmt.Errorf("failure getting reference source node: %v", err)
			} else if len(anchorNodeReply.Node) != 1 {
				return nil, fmt.Errorf("found %d nodes for {%+v}", len(anchorNodeReply.Node), anchor)
			} else if infoNodeKind(anchorNodeReply.Node[0]) != schema.AnchorKind {
				// Skip child if it isn't an anchor node
				continue
			}
			reply.Node = append(reply.Node, anchorNodeReply.Node[0])

			targets, err := getEdges(g.gs, anchor, func(e *spb.Entry) bool {
				return schema.EdgeDirection(e.GetEdgeKind()) == schema.Forward && e.GetEdgeKind() != schema.ChildOfEdge
			})
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve targets of anchor %v: %v", anchor, err)
			}
			if len(targets) == 0 {
				log.Printf("Anchor missing forward edges: {%+v}", anchor)
			}
			for _, edge := range targets {
				targetTicket := vnameToTicket(edge.Target)
				targetSet.Add(targetTicket)
				reply.Reference = append(reply.Reference, &xpb.DecorationsReply_Reference{
					SourceTicket: proto.String(vnameToTicket(anchor)),
					Kind:         proto.String(edge.Kind),
					TargetTicket: proto.String(targetTicket),
				})
			}
		}

		// Batch request all Reference target nodes
		nodesReply, err := g.Nodes(&xpb.NodesRequest{Ticket: targetSet.Slice()})
		if err != nil {
			return nil, fmt.Errorf("failure getting reference target nodes: %v", err)
		}
		reply.Node = append(reply.Node, nodesReply.Node...)
	}

	return reply, nil
}

var revChildOfEdgeKind = schema.MirrorEdge(schema.ChildOfEdge)

func infoNodeKind(info *xpb.NodeInfo) string {
	for _, fact := range info.Fact {
		if fact.GetName() == schema.NodeKindFact {
			return string(fact.Value)
		}
	}
	return ""
}

func getSourceText(gs storage.GraphStore, fileVName *spb.VName) (text []byte, encoding string, err error) {
	entries := make(chan *spb.Entry)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for entry := range entries {
			switch entry.GetFactName() {
			case schema.FileTextFact:
				text = entry.GetFactValue()
			case schema.FileEncodingFact:
				encoding = string(entry.GetFactValue())
			default:
				// skip other file facts
			}
		}
	}()

	err = gs.Read(&spb.ReadRequest{Source: fileVName}, entries)
	close(entries)
	if err != nil {
		return nil, "", fmt.Errorf("GraphStore error: %v", err)
	}
	wg.Wait()
	if text == nil {
		err = fmt.Errorf("file not found: %+v", fileVName)
	}
	return
}

type edgeTarget struct {
	Kind   string
	Target *spb.VName
}

// getEdges returns edgeTargets with the given node as their source.  Only edge
// entries that return true when applied to pred are returned.
func getEdges(gs storage.GraphStore, node *spb.VName, pred func(*spb.Entry) bool) ([]*edgeTarget, error) {
	var targets []*edgeTarget

	entries := make(chan *spb.Entry)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for entry := range entries {
			if entry.GetEdgeKind() != "" && pred(entry) {
				targets = append(targets, &edgeTarget{entry.GetEdgeKind(), entry.Target})
			}
		}
	}()

	err := gs.Read(&spb.ReadRequest{Source: node, EdgeKind: proto.String("*")}, entries)
	close(entries)
	if err != nil {
		return nil, fmt.Errorf("GraphStore error: %v", err)
	}
	wg.Wait()
	return targets, nil
}

func convertFilters(filters []string) []*regexp.Regexp {
	var patterns []*regexp.Regexp
	for _, filter := range filters {
		patterns = append(patterns, filterToRegexp(filter))
	}
	return patterns
}

var filterOpsRE = regexp.MustCompile("[*][*]|[*?]")

func filterToRegexp(pattern string) *regexp.Regexp {
	var re string
	for {
		loc := filterOpsRE.FindStringIndex(pattern)
		if loc == nil {
			break
		}
		re += regexp.QuoteMeta(pattern[:loc[0]])
		switch pattern[loc[0]:loc[1]] {
		case "**":
			re += ".*"
		case "*":
			re += "[^/]*"
		case "?":
			re += "[^/]"
		default:
			log.Fatal("Unknown filter operator: " + pattern[loc[0]:loc[1]])
		}
		pattern = pattern[loc[1]:]
	}
	return regexp.MustCompile(re + regexp.QuoteMeta(pattern))
}

func vnameToTicket(v *spb.VName) string {
	return kytheuri.FromVName(v).String()
}

func entryToFact(entry *spb.Entry) *xpb.Fact {
	return &xpb.Fact{
		Name:  entry.FactName,
		Value: entry.FactValue,
	}
}

// matchesAny reports whether if str matches any of the patterns
func matchesAny(str string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(str) {
			return true
		}
	}
	return false
}
