package goscore

import (
	"encoding/xml"
	"strconv"
)

type truePredicate struct{}
type dummyMiningSchema struct{}

// Node - PMML tree node
type Node struct {
	XMLName            xml.Name
	Attrs              []xml.Attr         `xml:",any,attr"`
	Content            []byte             `xml:",innerxml"`
	Nodes              []Node             `xml:",any"`
	True               truePredicate      `xml:"True"`
	DummyMiningSchema  dummyMiningSchema  `xml:"MiningSchema"`
	SimplePredicate    SimplePredicate    `xml:"SimplePredicate"`
	SimpleSetPredicate SimpleSetPredicate `xml:"SimpleSetPredicate"`
}

// TraverseTree - traverses Node predicates with features and returns score by terminal node
func (n Node) TraverseTree(features map[string]string) (score float64) {
	curr := n.Nodes[0]
	for len(curr.Nodes) > 0 {
		prevID := curr.Attrs[0].Value
		curr = step(curr, features)
		if prevID == curr.Attrs[0].Value {
			break
		}
	}
	score, _ = strconv.ParseFloat(curr.Attrs[1].Value, 64)
	return score
}

func step(curr Node, features map[string]string) Node {
	for _, node := range curr.Nodes {
		if node.XMLName.Local == "True" || node.SimplePredicate.True(features) || node.SimpleSetPredicate.True(features) {
			curr = node
			break
		}
	}
	return curr
}
