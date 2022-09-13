package util

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
)

type TextNodes struct {
	nodes       []*TextNode
	prefixNodes []*TextNode

	allNodesMap map[string]*TextNode
	isWindows   bool
	nextChildId int
	debug       bool
}

func NewTextNodes() *TextNodes {
	return &TextNodes{
		isWindows:   runtime.GOOS == "windows",
		allNodesMap: map[string]*TextNode{},
	}
}

type TextNode struct {
	ID                 string
	lines              []string
	requires           map[string]interface{}
	exports            map[string]interface{}
	directDependencies map[string]interface{}

	parent *TextNodes
}

func (tn *TextNodes) NewPrefixTextNode() *TextNode {

	res := &TextNode{
		parent:             tn,
		ID:                 tn.nextChildID(),
		requires:           map[string]interface{}{},
		exports:            map[string]interface{}{},
		directDependencies: map[string]interface{}{},
	}

	tn.prefixNodes = append(tn.prefixNodes, res)
	tn.allNodesMap[res.ID] = res

	return res
}

func (tn *TextNodes) NewTextNode() *TextNode {
	res := &TextNode{
		parent:             tn,
		ID:                 tn.nextChildID(),
		requires:           map[string]interface{}{},
		exports:            map[string]interface{}{},
		directDependencies: map[string]interface{}{},
	}

	tn.nodes = append(tn.nodes, res)
	tn.allNodesMap[res.ID] = res

	return res
}

func (textnode *TextNode) Out(str ...string) {
	if len(str) == 0 {
		str = []string{""}
	}

	textnode.lines = append(textnode.lines, str...)
}

func (textnode *TextNode) AddDependency(dependency *TextNode) {

	if textnode.parent.debug {
		fmt.Println(textnode.ID + " -> " + dependency.ID)
	}

	textnode.directDependencies[dependency.ID] = dependency.ID
}

func (textnode *TextNode) AddRequires(str ...string) {
	if len(str) == 0 {
		Panic("unexpected addRequires string length")
		return
	}

	for _, key := range str {
		textnode.requires[key] = ""
	}

}

func (textnode *TextNode) AddExports(str ...string) {
	if len(str) == 0 {
		Panic("unexpected addRequires string length")
		return
	}

	for _, key := range str {
		textnode.exports[key] = ""
	}
}

func (textnode *TextNode) SetEnv(envName string, value string) {
	if textnode.parent.isWindows {

		value = FixWindowsPathSuffix(value)
		textnode.Out(fmt.Sprintf("set %s=%s", envName, value))
	} else {
		// Export is used due to need to use 'bash -c (...)' at end of script
		textnode.Out(fmt.Sprintf("export %s=\"%s\"", envName, value))
	}

	textnode.AddExports(envName)
}

func (textnode *TextNode) Header(str string) {

	if !strings.HasSuffix(str, " ") {
		str += " "
	}

	for len(str) < 80 {
		str = str + "-"
	}

	if textnode.parent.IsWindows() {
		textnode.Out("REM " + str)
	} else {
		textnode.Out("# " + str)
	}

}

func (textnode *TextNode) ToString() string {
	output := ""

	for _, line := range textnode.lines {

		output += line

		if textnode.parent.isWindows {
			output += "\r\n"
		} else {
			output += "\n"
		}
	}

	return output
}

func (tn *TextNodes) IsWindows() bool {
	return tn.isWindows
}

func (tn TextNodes) ToString() (string, error) {
	var res string

	// Variable name -> Child text node id
	exportedVars := map[string]string{}
	{
		for _, childTextNode := range tn.allNodesMap {

			for exportedVar := range childTextNode.exports {

				if _, exists := exportedVars[exportedVar]; exists {
					return "", fmt.Errorf("multiple child text nodes are exporting: %v", exportedVar)
				}

				exportedVars[exportedVar] = childTextNode.ID

			}
		}
	}

	// text node id -> # of direct and indirect dependencies
	totalChildDependencies := map[string]int{}

	{
		// child text node id -> parent text node id
		childToParentMap := map[string][]string{}

		for _, childTextNode := range tn.allNodesMap {

			childDependencies := map[string]interface{}{}

			for directDependencyID := range childTextNode.directDependencies {
				childDependencies[directDependencyID] = directDependencyID
			}

			for requiredVar := range childTextNode.requires {

				parentNode, exists := exportedVars[requiredVar]
				if !exists {
					return "", fmt.Errorf("child text node requires unexported variable: %v", requiredVar)
				}

				childDependencies[parentNode] = parentNode
			}

			var allChildDependencies []string
			for parentID := range childDependencies {
				allChildDependencies = append(allChildDependencies, parentID)
			}

			childToParentMap[childTextNode.ID] = allChildDependencies
		}

		var findIndirectDependencies func(childID string) []string

		findIndirectDependencies = func(currID string) []string {

			resultMap := map[string]string{}

			for _, directDependency := range childToParentMap[currID] {

				dependendenciesOfDirectDependecy := findIndirectDependencies(directDependency)

				for _, dependendencyOfChildDependecy := range dependendenciesOfDirectDependecy {
					resultMap[dependendencyOfChildDependecy] = dependendencyOfChildDependecy
				}

			}

			keys := make([]string, 0, len(resultMap))
			for k := range resultMap {
				keys = append(keys, k)
			}

			return keys
		}

		for _, childTextNode := range tn.allNodesMap {
			totalChildDependencies[childTextNode.ID] = len(findIndirectDependencies(childTextNode.ID))
		}
	}

	sortFunc := func(textNodes []*TextNode) {
		sort.SliceStable(textNodes, func(i, j int) bool {

			var iDependencies, jDependencies int

			if totalDepends, exists := totalChildDependencies[textNodes[i].ID]; exists {
				iDependencies = totalDepends
			} else {
				iDependencies = 0
			}

			if totalDepends, exists := totalChildDependencies[textNodes[j].ID]; exists {
				jDependencies = totalDepends
			} else {
				jDependencies = 0
			}

			return iDependencies > jDependencies
		})
	}

	sortFunc(tn.prefixNodes)
	sortFunc(tn.nodes)

	for _, prefixNode := range tn.prefixNodes {
		res += prefixNode.ToString()
	}

	for _, node := range tn.nodes {
		res += node.ToString()
	}

	return res, nil
}

func Panic(err string) {
	fmt.Println(err)
	debug.PrintStack()
	os.Exit(1)
}

func (tn *TextNodes) nextChildID() string {
	id := fmt.Sprintf("%d", tn.nextChildId)
	tn.nextChildId++
	return id
}

// func (buffer *TextNode) Env(envName string) string {
// 	if buffer.isWindows {
// 		return "%" + envName + "%"
// 	} else {
// 		return "${" + envName + "}"
// 	}
// }
