package goscore

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type truePredicate struct {
}
type dummyMiningSchema struct{}

// Node - PMML tree node
type Node struct {
	XMLName            xml.Name
	Attrs              []xml.Attr         `xml:",any,attr"`
	Nodes              []Node             `xml:",any"`
	True               truePredicate      `xml:"True"`
	DummyMiningSchema  dummyMiningSchema  `xml:"MiningSchema"`
	SimplePredicate    SimplePredicate    `xml:"SimplePredicate"`
	SimpleSetPredicate SimpleSetPredicate `xml:"SimpleSetPredicate"`
}

// TraverseTree - traverses Node predicates with features and returns score by terminal node
func (n Node) TraverseTree(features map[string]interface{}) (score float64, err error) {
	curr := n.Nodes[0]
	cont := true

	for cont && len(curr.Nodes) > 0 {
		curr, cont = step(curr, features)
	}

	// TODO handle cases like nullPrediction
	if len(curr.Attrs) == 0 {
		return 0.0, nil
	}
	return strconv.ParseFloat(curr.Attrs[0].Value, 64)
}

func (n Node) String() string {
	return n.StringWtihDepth(0)
}

func (n Node) StringWtihDepth(d int) string {

	good_nodes := []Node{}
	for _, tree := range n.Nodes {
		if tree.XMLName.Local != "ScoreDistribution" {
			good_nodes = append(good_nodes, tree)
		}
	}

	str := ""
	if len(good_nodes) > 0 {
		str = fmt.Sprintf("Tree with %d nodes. Predicate: %s {class %s}.", len(good_nodes), n.SimplePredicate, n.XMLName.Local)
		for i, tree := range good_nodes {
			str = fmt.Sprintf("%s\n%s%d: %s", str, strings.Repeat("|  ", d), i, tree.StringWtihDepth(d+1))
		}
	} else {
		str = fmt.Sprintf("Terminal node. Score: %s. Predicate: %s {class %s}", n.Score(), n.SimplePredicate, n.XMLName.Local)
	}
	return str
}

func (n Node) Score() string {
	if len(n.Attrs) == 0 {
		return ""
	}
	for _, a := range n.Attrs {
		if a.Name.Local == "score" {
			return a.Value
		}
	}
	return ""
}

func step(curr Node, features map[string]interface{}) (Node, bool) {
	for _, node := range curr.Nodes {
		if len(node.Nodes) > 0 && node.Nodes[0].XMLName.Local == "True" {
			return node, true
		} else if node.SimplePredicate.True(features) || node.SimpleSetPredicate.True(features) {
			return node, true
		}
	}
	return curr, false
}
