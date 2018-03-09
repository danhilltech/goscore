package goscore

import (
	"encoding/xml"
	"strconv"
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
