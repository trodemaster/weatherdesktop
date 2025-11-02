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
	
	// Find condition value divs
	// We're looking for div elements with class "conditionValue"
	conditions := p.findConditionValues(doc)
	
	// Based on the original bash script:
	// div:nth-child(4) > div.conditionValue = East status
	// div:nth-child(5) > div.conditionValue = West status
	// div:nth-child(6) > div.conditionValue = Conditions text
	
	if len(conditions) >= 2 {
		status.East = strings.TrimSpace(conditions[0])
		status.West = strings.TrimSpace(conditions[1])
	}
	
	// Check if pass is closed
	if strings.Contains(status.East, "Closed") {
		status.IsClosed = true
	}
	if strings.Contains(status.West, "Closed") {
		status.IsClosed = true
	}
	
	// Get conditions text if closed
	if status.IsClosed && len(conditions) >= 3 {
		status.Conditions = p.cleanConditionsText(conditions[2])
	}
	
	return status, nil
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

