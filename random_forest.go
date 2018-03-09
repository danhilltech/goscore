package goscore

import (
	"encoding/xml"
	"io/ioutil"
	"strconv"
	"sync"
)

type DataField struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
}

type DerivedField struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	FieldRef FieldRef
}

type FieldRef struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
}

// RandomForest - PMML Random Forest
type RandomForest struct {
	XMLName       xml.Name
	Trees         []Node         `xml:"MiningModel>Segmentation>Segment>TreeModel"`
	DataFields    []DataField    `xml:"DataDictionary>DataField"`
	DerivedFields []DerivedField `xml:"TransformationDictionary>DerivedField"`
}

// LoadRandomForest - Load Random Forest PMML file to RandomForest struct
func LoadRandomForest(fileName string) (rf RandomForest, err error) {
	randomForestXml, err := ioutil.ReadFile(fileName)
	if err != nil {
		return rf, err
	}

	err = xml.Unmarshal([]byte(randomForestXml), &rf)
	if err != nil {
		return rf, err
	}
	return rf, nil
}

// Score - traverses all trees in RandomForest with features and returns ratio of
// given label results count to all results count
func (rf RandomForest) Score(features_public map[string]interface{}, label string) (float64, error) {
	features, err := rf.publicToDerivedFeatures(features_public)
	if err != nil {
		return 0.0, err
	}
	labelScores, err := rf.LabelScores(features)
	if err != nil {
		return 0.0, err
	}
	result := scoreByLabel(labelScores, label)
	return result, err
}

// ScoreConcurrently - same as Score but concurrent
func (rf RandomForest) ScoreConcurrently(features map[string]interface{}, label string) (float64, error) {
	labelScores, err := rf.LabelScoresConcurrently(features)
	result := scoreByLabel(labelScores, label)
	return result, err
}

func (rf RandomForest) Predict(features_public map[string]interface{}) (string, float64, error) {
	features, err := rf.publicToDerivedFeatures(features_public)
	if err != nil {
		return "", 0.0, err
	}
	labelScores, err := rf.LabelScores(features)
	if err != nil {
		return "", 0.0, err
	}

	winner := ""
	winnerVal := 0.0
	confidence := 0.0
	allCount := 0.0
	for _, value := range labelScores {
		allCount += value
	}

	for k, v := range labelScores {
		if v > winnerVal {
			winnerVal = v
			winner = k
			confidence = v / allCount

		}
	}
	return winner, confidence, nil
}

// LabelScores - traverses all trees in RandomForest with features and maps result
// labels to how many trees returned those label
func (rf RandomForest) LabelScores(features_public map[string]interface{}) (map[string]float64, error) {
	scores := map[string]float64{}
	features, err := rf.publicToDerivedFeatures(features_public)
	if err != nil {
		return scores, err
	}

	for _, tree := range rf.Trees {
		score, err := tree.TraverseTree(features)
		if err != nil {
			return scores, err
		}
		scoreString := strconv.FormatFloat(score, 'f', -1, 64)
		scores[scoreString]++
	}
	return scores, nil
}

// LabelScoresConcurrently - same as LabelScores but concurrent
func (rf RandomForest) LabelScoresConcurrently(features map[string]interface{}) (map[string]float64, error) {
	messages := rf.traverseConcurrently(features)
	return aggregateScores(messages, len(rf.Trees))
}

func (rf RandomForest) PublicFeatures() ([]string, error) {
	features := []string{}
	for _, field := range rf.DataFields {
		for _, a := range field.Attrs {
			if a.Name.Local == "name" {
				features = append(features, a.Value)
			}
		}
	}
	return features, nil
}

func (rf RandomForest) publicToDerivedFeature(public string) string {
	for _, field := range rf.DerivedFields {
		if field.FieldRef.Attrs[0].Value == public {
			for _, a := range field.Attrs {
				if a.Name.Local == "name" {
					return a.Value
				}
			}
		}
	}
	return public
}

func (rf RandomForest) publicToDerivedFeatures(features_public map[string]interface{}) (map[string]interface{}, error) {

	var features = make(map[string]interface{}, len(features_public))
	for k, v := range features_public {
		features[rf.publicToDerivedFeature(k)] = v
	}

	return features, nil
}

func aggregateScores(messages chan rfResult, treeCount int) (map[string]float64, error) {
	scores := map[string]float64{}
	var res rfResult
	for i := 0; i < treeCount; i++ {
		res = <-messages
		if res.ErrorName != nil {
			return map[string]float64{}, res.ErrorName
		}
		scores[res.Score]++
	}
	return scores, nil
}

func (rf RandomForest) traverseConcurrently(features map[string]interface{}) chan rfResult {
	messages := make(chan rfResult, len(rf.Trees))
	var wg sync.WaitGroup
	wg.Add(len(rf.Trees))
	for _, tree := range rf.Trees {
		go func(tree Node, features map[string]interface{}) {
			treeScore, err := tree.TraverseTree(features)
			scoreString := strconv.FormatFloat(treeScore, 'f', -1, 64)
			messages <- rfResult{ErrorName: err, Score: scoreString}
			wg.Done()
		}(tree, features)
	}
	wg.Wait()
	return messages
}

func scoreByLabel(labelScores map[string]float64, label string) float64 {
	allCount := 0.0
	for _, value := range labelScores {
		allCount += value
	}
	result := labelScores[label] / allCount
	return result
}

type rfResult struct {
	ErrorName error
	Score     string
}
