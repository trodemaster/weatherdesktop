package parser

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// PassStatus represents the status of a mountain pass
type PassStatus struct {
	East       string
	West       string
	IsClosed   bool
	Conditions string
}

// Parser handles HTML parsing
type Parser struct{}

// New creates a new parser
func New() *Parser {
	return &Parser{}
}

// ParseWSDOTPassStatus parses the WSDOT pass status HTML
func (p *Parser) ParseWSDOTPassStatus(htmlPath string) (*PassStatus, error) {
	// Read HTML file
	f, err := os.Open(htmlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open HTML file: %w", err)
	}
	defer f.Close()
	
	// Parse HTML
	doc, err := html.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	// Extract status information
	status := &PassStatus{
		East: "Open",
		West: "Open",
	}
	
	// Find all condition divs with their labels and values
	conditions := p.findConditionsWithLabels(doc)
	
	// Debug: log all conditions found
	// fmt.Printf("DEBUG: Found %d conditions\n", len(conditions))
	// for i, cond := range conditions {
	// 	fmt.Printf("DEBUG: Condition %d - Label: %q, Value: %q\n", i, cond.label, cond.value)
	// }
	
	// Look for "Travel eastbound" and "Travel westbound" labels
	for _, cond := range conditions {
		label := strings.ToLower(cond.label)
		value := strings.TrimSpace(cond.value)
		
		// Debug: log what we're finding
		// fmt.Printf("DEBUG: Found condition - Label: %q, Value: %q\n", cond.label, value)
		
		if strings.Contains(label, "travel") && strings.Contains(label, "eastbound") {
			status.East = value
			if strings.Contains(value, "Closed") || strings.Contains(value, "closed") {
				status.IsClosed = true
			}
		}
		if strings.Contains(label, "travel") && strings.Contains(label, "westbound") {
			status.West = value
			if strings.Contains(value, "Closed") || strings.Contains(value, "closed") {
				status.IsClosed = true
			}
		}
		if strings.Contains(label, "conditions") && status.IsClosed {
			status.Conditions = p.cleanConditionsText(value)
		}
	}
	
	
	return status, nil
}

// conditionPair represents a label-value pair
type conditionPair struct {
	label string
	value string
}

// findConditionsWithLabels finds conditionLabel and conditionValue pairs
// These are siblings within a parent div with class "condition"
func (p *Parser) findConditionsWithLabels(n *html.Node) []conditionPair {
	var results []conditionPair
	
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "div" {
			// Check if this is a parent div with class "condition"
			var isConditionParent bool
			for _, attr := range node.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "condition") {
					isConditionParent = true
					break
				}
			}
			
			if isConditionParent {
				// Look for conditionLabel and conditionValue children
				var label, value string
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					if child.Type == html.ElementNode && child.Data == "div" {
						for _, attr := range child.Attr {
							if attr.Key == "class" {
								if strings.Contains(attr.Val, "conditionLabel") {
									label = p.extractText(child)
								}
								if strings.Contains(attr.Val, "conditionValue") {
									value = p.extractText(child)
								}
							}
						}
					}
				}
				
				if label != "" && value != "" {
					results = append(results, conditionPair{
						label: strings.TrimSpace(label),
						value: strings.TrimSpace(value),
					})
				}
			}
		}
		
		// Traverse children
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	
	traverse(n)
	return results
}

// findConditionValues finds all text content from divs with class "conditionValue"
func (p *Parser) findConditionValues(n *html.Node) []string {
	var results []string
	
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "div" {
			// Check if this div has class "conditionValue"
			for _, attr := range node.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "conditionValue") {
					// Extract text content
					text := p.extractText(node)
					if text != "" {
						results = append(results, text)
					}
					break
				}
			}
		}
		
		// Traverse children
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	
	traverse(n)
	return results
}

// extractText extracts all text content from a node and its children
func (p *Parser) extractText(n *html.Node) string {
	var buf strings.Builder
	
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.TextNode {
			buf.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	
	traverse(n)
	return buf.String()
}

// cleanConditionsText cleans up the conditions text
// Replicates: sed 's/ \{2,\}/ /g' | tr -d '\n'
func (p *Parser) cleanConditionsText(text string) string {
	// Remove newlines
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	
	// Replace multiple spaces with single space
	re := regexp.MustCompile(`\s{2,}`)
	text = re.ReplaceAllString(text, " ")
	
	// Trim leading/trailing whitespace
	text = strings.TrimSpace(text)
	
	return text
}

