package adt

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// --- i18n Types ---

// DataElementLabels holds the text labels of a data element in a specific language.
type DataElementLabels struct {
	Short   string `json:"short" xml:"shortDescription,attr"`
	Medium  string `json:"medium" xml:"mediumDescription,attr"`
	Long    string `json:"long" xml:"longDescription,attr"`
	Heading string `json:"heading" xml:"heading,attr"`
}

// LanguageComparison holds the result of comparing an object's texts in two languages.
type LanguageComparison struct {
	SourceLang string            `json:"sourceLang"`
	TargetLang string            `json:"targetLang"`
	Entries    []ComparisonEntry `json:"entries"`
}

// ComparisonEntry represents a single text key compared between two languages.
type ComparisonEntry struct {
	Key        string `json:"key"`
	SourceText string `json:"sourceText"`
	TargetText string `json:"targetText"`
	Missing    bool   `json:"missing"`
}

// --- i18n Methods ---

// GetObjectTextsInLanguage retrieves the source/content of an object in a specific language.
// objectSourceURL is the ADT source URL (e.g., /sap/bc/adt/programs/programs/ZTEST/source/main).
func (c *Client) GetObjectTextsInLanguage(ctx context.Context, objectSourceURL, lang string) (string, error) {
	if err := c.checkSafety(OpRead, "GetObjectTextsInLanguage"); err != nil {
		return "", err
	}

	lang = strings.ToUpper(lang)

	resp, err := c.transport.Request(ctx, objectSourceURL, &RequestOptions{
		Method:           http.MethodGet,
		OverrideLanguage: lang,
	})
	if err != nil {
		return "", fmt.Errorf("get object texts in language %s: %w", lang, err)
	}

	return string(resp.Body), nil
}

// GetDataElementLabels retrieves the text labels of a data element in a specific language.
func (c *Client) GetDataElementLabels(ctx context.Context, name, lang string) (*DataElementLabels, error) {
	if err := c.checkSafety(OpRead, "GetDataElementLabels"); err != nil {
		return nil, err
	}

	name = strings.ToUpper(name)
	lang = strings.ToUpper(lang)

	path := fmt.Sprintf("/sap/bc/adt/ddic/dataelements/%s", url.PathEscape(name))
	resp, err := c.transport.Request(ctx, path, &RequestOptions{
		Method:           http.MethodGet,
		Accept:           "application/xml",
		OverrideLanguage: lang,
	})
	if err != nil {
		return nil, fmt.Errorf("get data element labels: %w", err)
	}

	// Parse the XML - data element labels are attributes on the root element
	var labels DataElementLabels
	if err := xml.Unmarshal(resp.Body, &labels); err != nil {
		return nil, fmt.Errorf("parse data element labels: %w", err)
	}

	return &labels, nil
}

// GetMessageClassTexts retrieves all messages of a message class in a specific language.
func (c *Client) GetMessageClassTexts(ctx context.Context, name, lang string) ([]MessageClassMessage, error) {
	if err := c.checkSafety(OpRead, "GetMessageClassTexts"); err != nil {
		return nil, err
	}

	name = strings.ToUpper(name)
	lang = strings.ToUpper(lang)

	path := fmt.Sprintf("/sap/bc/adt/messageclass/%s", url.PathEscape(strings.ToLower(name)))
	resp, err := c.transport.Request(ctx, path, &RequestOptions{
		Method:           http.MethodGet,
		Accept:           "application/vnd.sap.adt.mc.messageclass+xml",
		OverrideLanguage: lang,
	})
	if err != nil {
		return nil, fmt.Errorf("get message class texts: %w", err)
	}

	var mc MessageClass
	if err := xml.Unmarshal(resp.Body, &mc); err != nil {
		return nil, fmt.Errorf("parse message class XML: %w", err)
	}

	return mc.Messages, nil
}

// WriteMessageClassTexts updates message class texts in a specific language.
// Requires a lock handle from LockObject and optionally a transport request number.
func (c *Client) WriteMessageClassTexts(ctx context.Context, name, lang string, texts []MessageClassMessage, lockHandle, transport string) error {
	name = strings.ToUpper(name)
	lang = strings.ToUpper(lang)

	// Unified mutation policy gate (op type + package + transport)
	if err := c.checkMutation(ctx, MutationContext{
		Op:        OpUpdate,
		OpName:    "WriteMessageClassTexts",
		ObjectURL: fmt.Sprintf("/sap/bc/adt/messageclass/%s", url.PathEscape(strings.ToLower(name))),
		Transport: transport,
	}); err != nil {
		return err
	}

	// Build XML body
	mc := MessageClass{
		Name:     name,
		Messages: texts,
	}
	body, err := xml.Marshal(mc)
	if err != nil {
		return fmt.Errorf("marshal message class XML: %w", err)
	}

	path := fmt.Sprintf("/sap/bc/adt/messageclass/%s", url.PathEscape(strings.ToLower(name)))

	params := url.Values{}
	params.Set("lockHandle", lockHandle)
	if transport != "" {
		params.Set("corrNr", transport)
	}

	_, err = c.transport.Request(ctx, path, &RequestOptions{
		Method:           http.MethodPut,
		Query:            params,
		Body:             body,
		ContentType:      "application/vnd.sap.adt.mc.messageclass+xml",
		OverrideLanguage: lang,
	})
	if err != nil {
		return fmt.Errorf("write message class texts: %w", err)
	}

	return nil
}

// WriteDataElementLabels updates data element labels in a specific language.
// Requires a lock handle from LockObject and optionally a transport request number.
func (c *Client) WriteDataElementLabels(ctx context.Context, name, lang string, labels *DataElementLabels, lockHandle, transport string) error {
	name = strings.ToUpper(name)
	lang = strings.ToUpper(lang)

	// Unified mutation policy gate (op type + package + transport)
	if err := c.checkMutation(ctx, MutationContext{
		Op:        OpUpdate,
		OpName:    "WriteDataElementLabels",
		ObjectURL: fmt.Sprintf("/sap/bc/adt/ddic/dataelements/%s", url.PathEscape(name)),
		Transport: transport,
	}); err != nil {
		return err
	}

	body, err := xml.Marshal(labels)
	if err != nil {
		return fmt.Errorf("marshal data element labels: %w", err)
	}

	path := fmt.Sprintf("/sap/bc/adt/ddic/dataelements/%s", url.PathEscape(name))

	params := url.Values{}
	params.Set("lockHandle", lockHandle)
	if transport != "" {
		params.Set("corrNr", transport)
	}

	_, err = c.transport.Request(ctx, path, &RequestOptions{
		Method:           http.MethodPut,
		Query:            params,
		Body:             body,
		ContentType:      "application/xml",
		OverrideLanguage: lang,
	})
	if err != nil {
		return fmt.Errorf("write data element labels: %w", err)
	}

	return nil
}

// GetTextPoolInLanguage retrieves a program's text elements (text symbols /
// selection texts / list headings) in a specific language. If category is empty
// all three categories (symbols, selections, headings) are read and merged;
// otherwise only the requested category is returned. Each entry carries its
// Category. Mirrors abap-adt-api getTextElements (plain-text protocol: a GET to
// /textelements/programs/<name>/source/<category> with Accept
// application/vnd.sap.adt.textelements.<category>.v1 returns a plain-text body
// of "ID=text" lines). A 404 for a category is treated as "no entries".
func (c *Client) GetTextPoolInLanguage(ctx context.Context, programName, category, lang string) ([]TextElement, error) {
	if err := c.checkSafety(OpRead, "GetTextPoolInLanguage"); err != nil {
		return nil, err
	}

	programName = strings.ToUpper(programName)
	lang = strings.ToUpper(lang)
	category = strings.ToLower(strings.TrimSpace(category))
	if category != "" && !validTextElementCategories[category] {
		return nil, fmt.Errorf("invalid text element category %q: want one of symbols, selections, headings", category)
	}

	cats := []string{category}
	if category == "" {
		cats = []string{"symbols", "selections", "headings"}
	}

	var all []TextElement
	for _, cat := range cats {
		entries, err := c.readTextElementsCategory(ctx, programName, cat, lang)
		if err != nil {
			return nil, err
		}
		for i := range entries {
			entries[i].Category = cat
		}
		all = append(all, entries...)
	}
	return all, nil
}

// readTextElementsCategory fetches one text-element category for a program as the
// plain-text body described above; a 404 yields an empty (nil) result.
func (c *Client) readTextElementsCategory(ctx context.Context, programName, category, lang string) ([]TextElement, error) {
	path := fmt.Sprintf("/sap/bc/adt/textelements/programs/%s/source/%s", url.PathEscape(programName), category)
	resp, err := c.transport.Request(ctx, path, &RequestOptions{
		Method:           http.MethodGet,
		Accept:           fmt.Sprintf("application/vnd.sap.adt.textelements.%s.v1", category),
		OverrideLanguage: lang,
	})
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return nil, nil
		}
		return nil, fmt.Errorf("get text pool (%s): %w", category, err)
	}
	return parseTextElements(string(resp.Body)), nil
}

// parseTextElements parses the plain-text body returned by the ADT textelements
// endpoint: one "ID=text" entry per line, optionally preceded by "@MaxLength:N"
// and/or "@DDICReference:ref" prefix lines that apply to the next entry. Mirrors
// abap-adt-api parseTextElements.
func parseTextElements(body string) []TextElement {
	var elements []TextElement
	var curMaxLength int
	var curDdicRef string
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "@MaxLength:"):
			if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "@MaxLength:"))); err == nil {
				curMaxLength = n
			}
		case strings.HasPrefix(line, "@DDICReference:"):
			curDdicRef = strings.TrimSpace(strings.TrimPrefix(line, "@DDICReference:"))
		case strings.Contains(line, "="):
			eq := strings.Index(line, "=")
			id := strings.TrimSpace(line[:eq])
			text := line[eq+1:]
			if id != "" {
				el := TextElement{ID: id, Text: text}
				if curMaxLength > 0 {
					el.MaxLength = curMaxLength
				}
				if curDdicRef != "" {
					el.DDicReference = curDdicRef
				}
				elements = append(elements, el)
			}
			curMaxLength = 0
			curDdicRef = ""
		}
	}
	return elements
}

// CompareObjectLanguages compares the text content of an object in two languages.
// Returns a comparison showing which texts differ or are missing in the target language.
func (c *Client) CompareObjectLanguages(ctx context.Context, objectSourceURL, sourceLang, targetLang string) (*LanguageComparison, error) {
	if err := c.checkSafety(OpRead, "CompareObjectLanguages"); err != nil {
		return nil, err
	}

	// Get source language content
	sourceContent, err := c.GetObjectTextsInLanguage(ctx, objectSourceURL, sourceLang)
	if err != nil {
		return nil, fmt.Errorf("get source language (%s): %w", sourceLang, err)
	}

	// Get target language content
	targetContent, err := c.GetObjectTextsInLanguage(ctx, objectSourceURL, targetLang)
	if err != nil {
		return nil, fmt.Errorf("get target language (%s): %w", targetLang, err)
	}

	// Build comparison by splitting into lines
	sourceLines := strings.Split(sourceContent, "\n")
	targetLines := strings.Split(targetContent, "\n")

	// Build target map for lookup
	targetMap := make(map[int]string)
	for i, line := range targetLines {
		targetMap[i] = line
	}

	comparison := &LanguageComparison{
		SourceLang: strings.ToUpper(sourceLang),
		TargetLang: strings.ToUpper(targetLang),
	}

	for i, sourceLine := range sourceLines {
		entry := ComparisonEntry{
			Key:        fmt.Sprintf("line-%d", i+1),
			SourceText: sourceLine,
		}
		if targetLine, ok := targetMap[i]; ok {
			entry.TargetText = targetLine
			entry.Missing = false
		} else {
			entry.Missing = true
		}
		if entry.SourceText != entry.TargetText || entry.Missing {
			comparison.Entries = append(comparison.Entries, entry)
		}
	}

	return comparison, nil
}
