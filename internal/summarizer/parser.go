package summarizer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/logger"
)

// MetadataParser parses summary text to extract structured metadata
type MetadataParser struct{}

// SummaryMetadata contains extracted metadata from a summary
type SummaryMetadata struct {
	Sentiment        string
	CredibilityScore int
	Products         []db.ProductMention
	ProductsJSON     string
	RedFlagsCount    int
	ValidationStatus string
}

// NewMetadataParser creates a new metadata parser
func NewMetadataParser() *MetadataParser {
	return &MetadataParser{}
}

// Parse extracts all metadata from summary text
func (p *MetadataParser) Parse(summaryText string) SummaryMetadata {
	logger.Debug("Parsing summary metadata...")
	
	metadata := SummaryMetadata{}
	
	// Extract sentiment
	metadata.Sentiment = p.extractSentiment(summaryText)
	
	// Extract products
	metadata.Products = p.extractProducts(summaryText)
	
	// Convert products to JSON
	productsJSON, err := json.Marshal(extractProductNames(metadata.Products))
	if err != nil {
		logger.Error("Failed to marshal products to JSON: %v", err)
		metadata.ProductsJSON = "[]"
	} else {
		metadata.ProductsJSON = string(productsJSON)
	}
	
	// Calculate overall credibility
	metadata.CredibilityScore = p.calculateCredibility(metadata.Products, summaryText)
	
	// Count red flags
	metadata.RedFlagsCount = p.countRedFlags(summaryText)
	
	// Determine validation status
	metadata.ValidationStatus = p.determineStatus(metadata.CredibilityScore, metadata.RedFlagsCount)
	
	logger.Info("✅ Metadata parsed: Sentiment=%s, Credibility=%d/5, Products=%d, RedFlags=%d, Status=%s",
		metadata.Sentiment, metadata.CredibilityScore, len(metadata.Products), 
		metadata.RedFlagsCount, metadata.ValidationStatus)
	
	return metadata
}

// extractSentiment extracts sentiment from summary text
func (p *MetadataParser) extractSentiment(text string) string {
	// Look for sentiment patterns in Indonesian
	// "Sentiment umum: positive/positif" or "Sentiment harian: negative/negatif"
	
	// Try various patterns
	patterns := []string{
		`Sentiment\s+(?:umum|harian):\s*(\w+)`,
		`Sentiment\s*:\s*(\w+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			sentiment := strings.ToLower(strings.TrimSpace(matches[1]))
			
			// Normalize sentiment
			if strings.Contains(sentiment, "positif") || strings.Contains(sentiment, "positive") {
				return "positive"
			} else if strings.Contains(sentiment, "negatif") || strings.Contains(sentiment, "negative") {
				return "negative"
			} else if strings.Contains(sentiment, "netral") || strings.Contains(sentiment, "neutral") {
				return "neutral"
			}
			
			return sentiment
		}
	}
	
	// Default to neutral if not found
	logger.Debug("Sentiment not found in text, defaulting to neutral")
	return "neutral"
}

// extractProducts extracts product mentions from summary text
func (p *MetadataParser) extractProducts(text string) []db.ProductMention {
	products := []db.ProductMention{}
	
	// Find the product section: "## 📦 PAKET/PRODUK YANG DIBAHAS" or "## 📦 PRODUK/PAKET YANG DIBAHAS"
	productSectionPattern := `(?s)##\s*📦\s*(?:PAKET/PRODUK|PRODUK/PAKET).*?(?:##|$)`
	re := regexp.MustCompile(productSectionPattern)
	productSection := re.FindString(text)
	
	if productSection == "" {
		logger.Debug("No product section found in summary")
		return products
	}
	
	// Extract individual products
	// Pattern: **[Product Name]**
	productNamePattern := `\*\*([^*]+)\*\*`
	re = regexp.MustCompile(productNamePattern)
	productNames := re.FindAllStringSubmatch(productSection, -1)
	
	for _, match := range productNames {
		if len(match) > 1 {
			productName := strings.TrimSpace(match[1])
			
			// Skip section headers
			if strings.Contains(productName, "Testimoni") || 
			   strings.Contains(productName, "Konsensus") ||
			   strings.Contains(productName, "Analisa") {
				continue
			}
			
			// Extract details for this product
			product := p.extractProductDetails(productName, text)
			products = append(products, product)
		}
	}
	
	logger.Debug("Extracted %d products from summary", len(products))
	return products
}

// extractProductDetails extracts detailed information about a specific product
func (p *MetadataParser) extractProductDetails(productName, text string) db.ProductMention {
	product := db.ProductMention{
		ProductName: productName,
	}
	
	// Find the product's detail section
	// Look for text between product name and next product or section
	escapedName := regexp.QuoteMeta(productName)
	pattern := fmt.Sprintf(`\*\*%s\*\*\s*(.*?)(?:\*\*[^*]+\*\*|##|$)`, escapedName)
	re := regexp.MustCompile(`(?s)` + pattern)
	matches := re.FindStringSubmatch(text)
	
	if len(matches) > 1 {
		details := matches[1]
		
		// Extract mention count: "Jumlah mention: 5 kali"
		mentionPattern := `(?:Jumlah\s+)?mention:\s*(\d+)\s*kali`
		re = regexp.MustCompile(`(?i)` + mentionPattern)
		mentionMatches := re.FindStringSubmatch(details)
		if len(mentionMatches) > 1 {
			fmt.Sscanf(mentionMatches[1], "%d", &product.MentionCount)
		}
		
		// Extract price: "Harga: Rp 30.000" or "Rp 50.000"
		pricePattern := `(?:Harga|Price):\s*((?:Rp\s*)?[\d.,]+)`
		re = regexp.MustCompile(`(?i)` + pricePattern)
		priceMatches := re.FindStringSubmatch(details)
		if len(priceMatches) > 1 {
			product.PriceMentioned = strings.TrimSpace(priceMatches[1])
		}
		
		// Extract credibility: "Rating kredibilitas: High" or "⭐⭐⭐⭐⭐"
		product.CredibilityScore = p.extractCredibilityScore(details)
		
		// Extract sentiment from context
		product.Sentiment = p.extractProductSentiment(details)
		
		// Determine validation status
		product.ValidationStatus = p.extractValidationStatus(productName, text)
	}
	
	// Default values if not found
	if product.MentionCount == 0 {
		product.MentionCount = 1 // At least mentioned once
	}
	if product.CredibilityScore == 0 {
		product.CredibilityScore = 3 // Default to medium
	}
	if product.Sentiment == "" {
		product.Sentiment = "neutral"
	}
	if product.ValidationStatus == "" {
		product.ValidationStatus = "mixed"
	}
	
	return product
}

// extractCredibilityScore extracts credibility rating from text
func (p *MetadataParser) extractCredibilityScore(text string) int {
	// Try star rating: ⭐⭐⭐⭐⭐
	stars := strings.Count(text, "⭐")
	if stars > 0 && stars <= 5 {
		return stars
	}
	
	// Try text rating: "High", "Medium", "Low"
	text = strings.ToLower(text)
	if strings.Contains(text, "high") || strings.Contains(text, "tinggi") {
		return 5
	} else if strings.Contains(text, "medium") || strings.Contains(text, "sedang") {
		return 3
	} else if strings.Contains(text, "low") || strings.Contains(text, "rendah") {
		return 1
	}
	
	return 3 // Default medium
}

// extractProductSentiment extracts sentiment for a specific product
func (p *MetadataParser) extractProductSentiment(text string) string {
	text = strings.ToLower(text)
	
	// Check for positive indicators
	positiveWords := []string{"positif", "positive", "bagus", "recommended", "mantap", "oke", "good"}
	for _, word := range positiveWords {
		if strings.Contains(text, word) {
			return "positive"
		}
	}
	
	// Check for negative indicators
	negativeWords := []string{"negatif", "negative", "jelek", "buruk", "tidak", "bad", "komplain", "complaint"}
	for _, word := range negativeWords {
		if strings.Contains(text, word) {
			return "negative"
		}
	}
	
	return "neutral"
}

// extractValidationStatus determines if product mention is valid or suspicious
func (p *MetadataParser) extractValidationStatus(productName, text string) string {
	// Look in validation section
	validSection := `(?s)##\s*✅\s*VALIDASI.*?(?:##|$)`
	re := regexp.MustCompile(validSection)
	validationSection := re.FindString(text)
	
	if validationSection == "" {
		return "mixed"
	}
	
	// Check if product is mentioned as valid
	if strings.Contains(validationSection, productName) {
		if strings.Contains(validationSection, "Valid") || strings.Contains(validationSection, "Trustworthy") {
			return "valid"
		}
		if strings.Contains(validationSection, "Meragukan") || strings.Contains(validationSection, "Suspicious") {
			return "suspicious"
		}
	}
	
	return "mixed"
}

// calculateCredibility calculates overall credibility score
func (p *MetadataParser) calculateCredibility(products []db.ProductMention, text string) int {
	if len(products) == 0 {
		// No products, check validation section
		return p.extractOverallCredibility(text)
	}
	
	// Average credibility of all products
	total := 0
	for _, product := range products {
		total += product.CredibilityScore
	}
	
	avg := total / len(products)
	
	// Adjust based on validation section
	overallCred := p.extractOverallCredibility(text)
	if overallCred > 0 {
		// Weight average: 70% products, 30% overall
		avg = (avg*7 + overallCred*3) / 10
	}
	
	// Ensure 1-5 range
	if avg < 1 {
		avg = 1
	}
	if avg > 5 {
		avg = 5
	}
	
	return avg
}

// extractOverallCredibility extracts overall credibility from validation section
func (p *MetadataParser) extractOverallCredibility(text string) int {
	// Count valid vs suspicious items
	validCount := strings.Count(text, "✅ VALID")
	suspiciousCount := strings.Count(text, "❌ SUSPICIOUS")
	mixedCount := strings.Count(text, "⚠️ MIXED")
	
	if validCount > suspiciousCount {
		return 5 // High credibility
	} else if suspiciousCount > validCount {
		return 1 // Low credibility
	} else if mixedCount > 0 || validCount == suspiciousCount {
		return 3 // Medium credibility
	}
	
	return 3 // Default medium
}

// countRedFlags counts red flags in the summary
func (p *MetadataParser) countRedFlags(text string) int {
	// Find red flags section: "## 🚩 RED FLAGS"
	redFlagPattern := `(?s)##\s*🚩\s*RED FLAGS.*?(?:##|$)`
	re := regexp.MustCompile(redFlagPattern)
	redFlagSection := re.FindString(text)
	
	if redFlagSection == "" {
		return 0
	}
	
	// Check if it says "no red flags"
	noRedFlagsPatterns := []string{
		"tidak ada red flags",
		"tidak ada propaganda",
		"no red flags",
		"none detected",
		"tidak terdeteksi",
	}
	
	lowerSection := strings.ToLower(redFlagSection)
	for _, pattern := range noRedFlagsPatterns {
		if strings.Contains(lowerSection, pattern) {
			return 0
		}
	}
	
	// Count bullet points or numbered items in red flags section
	bulletCount := strings.Count(redFlagSection, "- ")
	bulletCount += strings.Count(redFlagSection, "* ")
	bulletCount += strings.Count(redFlagSection, "• ")
	
	// Count numbered items
	for i := 1; i <= 10; i++ {
		pattern := fmt.Sprintf("%d. ", i)
		if strings.Contains(redFlagSection, pattern) {
			bulletCount++
		}
	}
	
	return bulletCount
}

// determineStatus determines overall validation status
func (p *MetadataParser) determineStatus(credibility, redFlags int) string {
	// High red flags = suspicious
	if redFlags >= 3 {
		return "suspicious"
	}
	
	// High credibility + low red flags = valid
	if credibility >= 4 && redFlags <= 1 {
		return "valid"
	}
	
	// Low credibility = suspicious
	if credibility <= 2 {
		return "suspicious"
	}
	
	// Everything else = mixed
	return "mixed"
}

// Helper function to extract product names only
func extractProductNames(products []db.ProductMention) []string {
	names := make([]string, len(products))
	for i, p := range products {
		names[i] = p.ProductName
	}
	return names
}
