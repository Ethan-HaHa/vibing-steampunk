package adt

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// --- Workflow Tools ---
// These tools combine multiple operations into atomic workflows for simpler usage.

// WriteProgramResult represents the result of writing a program.
type WriteProgramResult struct {
	Success      bool                       `json:"success"`
	ProgramName  string                     `json:"programName"`
	ObjectURL    string                     `json:"objectUrl"`
	SyntaxErrors []SyntaxCheckResult        `json:"syntaxErrors,omitempty"`
	Activation   *ActivationResult          `json:"activation,omitempty"`
	Message      string                     `json:"message,omitempty"`
}

// WriteProgram performs Lock -> SyntaxCheck -> UpdateSource -> Unlock -> Activate workflow.
// This is a convenience method for updating existing programs.
func (c *Client) WriteProgram(ctx context.Context, programName string, source string, transport string) (*WriteProgramResult, error) {
	programName = strings.ToUpper(programName)
	objectURL := fmt.Sprintf("/sap/bc/adt/programs/programs/%s", url.PathEscape(programName))
	sourceURL := objectURL + "/source/main"

	// Unified mutation policy gate (op type + package + transport)
	if err := c.checkMutation(ctx, MutationContext{
		Op:        OpWorkflow,
		OpName:    "WriteProgram",
		ObjectURL: objectURL,
		Transport: transport,
	}); err != nil {
		return nil, err
	}

	result := &WriteProgramResult{
		ProgramName: programName,
		ObjectURL:   objectURL,
	}

	// Step 1: Syntax check before making changes
	syntaxErrors, err := c.SyntaxCheck(ctx, objectURL, source)
	if err != nil {
		result.Message = fmt.Sprintf("Syntax check failed: %v", err)
		return result, nil
	}

	// Check for syntax errors
	for _, se := range syntaxErrors {
		if se.Severity == "E" || se.Severity == "A" || se.Severity == "X" {
			result.SyntaxErrors = syntaxErrors
			result.Message = "Source has syntax errors - not saved"
			return result, nil
		}
	}
	result.SyntaxErrors = syntaxErrors // Include warnings if any

	// Step 2: Lock the object
	lock, err := c.LockObject(ctx, objectURL, "MODIFY")
	if err != nil {
		result.Message = fmt.Sprintf("Failed to lock object: %v", err)
		return result, nil
	}

	// Ensure we unlock on any error
	defer func() {
		if !result.Success {
			c.UnlockObject(ctx, objectURL, lock.LockHandle)
		}
	}()

	// Step 3: Update source
	err = c.UpdateSource(ctx, sourceURL, source, lock.LockHandle, transport)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to update source: %v", err)
		return result, nil
	}

	// Step 4: Unlock before activation (SAP requirement)
	err = c.UnlockObject(ctx, objectURL, lock.LockHandle)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to unlock object: %v", err)
		return result, nil
	}

	// Step 5: Activate
	activation, err := c.Activate(ctx, objectURL, programName)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to activate: %v", err)
		result.Activation = activation
		return result, nil
	}

	result.Activation = activation
	if activation.Success {
		result.Success = true
		result.Message = "Program updated and activated successfully"
	} else {
		result.Message = "Activation failed - check activation messages"
	}

	return result, nil
}

// WriteClassResult represents the result of writing a class.
type WriteClassResult struct {
	Success      bool                       `json:"success"`
	ClassName    string                     `json:"className"`
	ObjectURL    string                     `json:"objectUrl"`
	SyntaxErrors []SyntaxCheckResult        `json:"syntaxErrors,omitempty"`
	Activation   *ActivationResult          `json:"activation,omitempty"`
	Message      string                     `json:"message,omitempty"`
}

// WriteClass performs Lock -> SyntaxCheck -> UpdateSource -> Unlock -> Activate workflow for classes.
func (c *Client) WriteClass(ctx context.Context, className string, source string, transport string) (*WriteClassResult, error) {
	className = strings.ToUpper(className)
	objectURL := fmt.Sprintf("/sap/bc/adt/oo/classes/%s", url.PathEscape(className))
	sourceURL := objectURL + "/source/main"

	// Unified mutation policy gate (op type + package + transport)
	if err := c.checkMutation(ctx, MutationContext{
		Op:        OpWorkflow,
		OpName:    "WriteClass",
		ObjectURL: objectURL,
		Transport: transport,
	}); err != nil {
		return nil, err
	}

	result := &WriteClassResult{
		ClassName: className,
		ObjectURL: objectURL,
	}

	// Step 1: Syntax check
	syntaxErrors, err := c.SyntaxCheck(ctx, objectURL, source)
	if err != nil {
		result.Message = fmt.Sprintf("Syntax check failed: %v", err)
		return result, nil
	}

	// Check for syntax errors
	for _, se := range syntaxErrors {
		if se.Severity == "E" || se.Severity == "A" || se.Severity == "X" {
			result.SyntaxErrors = syntaxErrors
			result.Message = "Source has syntax errors - not saved"
			return result, nil
		}
	}
	result.SyntaxErrors = syntaxErrors

	// Step 2: Lock
	lock, err := c.LockObject(ctx, objectURL, "MODIFY")
	if err != nil {
		result.Message = fmt.Sprintf("Failed to lock object: %v", err)
		return result, nil
	}

	defer func() {
		if !result.Success {
			c.UnlockObject(ctx, objectURL, lock.LockHandle)
		}
	}()

	// Step 3: Update source
	err = c.UpdateSource(ctx, sourceURL, source, lock.LockHandle, transport)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to update source: %v", err)
		return result, nil
	}

	// Step 4: Unlock
	err = c.UnlockObject(ctx, objectURL, lock.LockHandle)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to unlock object: %v", err)
		return result, nil
	}

	// Step 5: Activate
	activation, err := c.Activate(ctx, objectURL, className)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to activate: %v", err)
		result.Activation = activation
		return result, nil
	}

	result.Activation = activation
	if activation.Success {
		result.Success = true
		result.Message = "Class updated and activated successfully"
	} else {
		result.Message = "Activation failed - check activation messages"
	}

	return result, nil
}

// --- Text Pool (Text Elements) Workflow ---

// TextElement represents a single text element (text symbol / selection text /
// list heading) to write into a program's text pool. Mirrors abap-adt-api's
// TextElement (D:\GitHub\abap-adt-api\src\api\textelements.ts).
type TextElement struct {
	ID            string `json:"id"`
	Text          string `json:"text"`
	MaxLength     int    `json:"max_length,omitempty"`
	DDicReference string `json:"ddic_reference,omitempty"`
	Category      string `json:"category,omitempty"`
}

// WriteTextPoolResult represents the result of writing a program's text pool.
type WriteTextPoolResult struct {
	Success     bool              `json:"success"`
	ProgramName string            `json:"programName"`
	Category    string            `json:"category"`
	ObjectURL   string            `json:"objectUrl"`
	Activation  *ActivationResult `json:"activation,omitempty"`
	Message     string            `json:"message,omitempty"`
}

// validTextElementCategories is the set of ADT text element categories.
var validTextElementCategories = map[string]bool{
	"symbols":    true,
	"selections": true,
	"headings":   true,
}

// validHeadingKeys is the set of allowed IDs for the "headings" category.
var validHeadingKeys = map[string]bool{
	"LISTHEADER":     true,
	"COLUMNHEADER_1": true,
	"COLUMNHEADER_2": true,
	"COLUMNHEADER_3": true,
	"COLUMNHEADER_4": true,
}

// formatTextElements validates a set of text elements for a category and renders
// the plain-text body (`ID=text` per line) expected by the ADT textelements PUT.
// Mirrors abap-adt-api's formatTextElements / validateTextElements.
func formatTextElements(entries []TextElement, category string) (string, error) {
	if !validTextElementCategories[category] {
		return "", fmt.Errorf("invalid text element category %q: want one of symbols, selections, headings", category)
	}
	var lines []string
	for _, el := range entries {
		id := strings.ToUpper(strings.TrimSpace(el.ID))
		switch category {
		case "symbols":
			if len(id) != 3 {
				return "", fmt.Errorf("symbol key %q must be exactly 3 characters", el.ID)
			}
			if strings.ContainsAny(id, " \t\r\n") {
				return "", fmt.Errorf("symbol key %q must not contain blanks", el.ID)
			}
			// SAP rejects text symbols without an explicit @MaxLength with HTTP
			// 406 "Text elements contain errors; correct all inconsistencies"
			// (verified empirically — a single symbol without @MaxLength fails
			// even when the rest of the pool is valid). Default to the text
			// length in characters when the caller omits max_length, mirroring
			// SE38's text element editor which derives the length from the text
			// on save.
			textLen := len([]rune(el.Text))
			maxLen := el.MaxLength
			if maxLen <= 0 {
				maxLen = textLen
			}
			if textLen > maxLen {
				return "", fmt.Errorf("symbol %q text length %d exceeds maxLength %d", el.ID, textLen, maxLen)
			}
			lines = append(lines, fmt.Sprintf("@MaxLength:%d", maxLen))
		case "selections":
			if len(el.Text) > 30 {
				return "", fmt.Errorf("selection %q text length %d exceeds maximum of 30", el.ID, len(el.Text))
			}
			if el.DDicReference != "" {
				lines = append(lines, fmt.Sprintf("@DDICReference:%s", el.DDicReference))
			}
		case "headings":
			if !validHeadingKeys[id] {
				return "", fmt.Errorf("invalid heading key %q: allowed LISTHEADER, COLUMNHEADER_1..4", el.ID)
			}
			limit := 255
			if id == "LISTHEADER" {
				limit = 71
			}
			if len(el.Text) > limit {
				return "", fmt.Errorf("heading %q text length %d exceeds maximum of %d", el.ID, len(el.Text), limit)
			}
		}
		lines = append(lines, fmt.Sprintf("%s=%s", id, el.Text))
		if category != "headings" {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n"), nil
}

// WriteTextPool writes a program's text elements (text symbols / selection texts /
// list headings) for a category as a single workflow: Lock -> PUT -> Unlock ->
// Activate. The PUT uses the standard ADT textelements endpoint with a plain-text
// body and media type application/vnd.sap.adt.textelements.<category>.v1
// (protocol mirrors abap-adt-api setTextElements). The PUT is stateful so it stays
// within the lock session (issue #88).
func (c *Client) WriteTextPool(ctx context.Context, programName, category string, entries []TextElement, lang, transport string) (*WriteTextPoolResult, error) {
	programName = strings.ToUpper(programName)
	category = strings.ToLower(strings.TrimSpace(category))
	if lang != "" {
		lang = strings.ToUpper(lang)
	}

	lockURL := fmt.Sprintf("/sap/bc/adt/textelements/programs/%s/source/%s", url.PathEscape(programName), category)
	programURL := fmt.Sprintf("/sap/bc/adt/programs/programs/%s", url.PathEscape(programName))

	// Unified mutation policy gate (op type + package + transport)
	if err := c.checkMutation(ctx, MutationContext{
		Op:        OpWorkflow,
		OpName:    "WriteTextPool",
		ObjectURL: programURL,
		Transport: transport,
	}); err != nil {
		return nil, err
	}

	result := &WriteTextPoolResult{
		ProgramName: programName,
		Category:    category,
		ObjectURL:   lockURL,
	}

	// Render + validate the plain-text body up front (cheap, fail before locking)
	body, err := formatTextElements(entries, category)
	if err != nil {
		result.Message = fmt.Sprintf("Invalid text elements: %v", err)
		return result, nil
	}

	// Step 1: Lock the text elements resource
	lock, err := c.LockObject(ctx, lockURL, "MODIFY")
	if err != nil {
		result.Message = fmt.Sprintf("Failed to lock text elements: %v", err)
		return result, nil
	}

	defer func() {
		if !result.Success {
			c.UnlockObject(ctx, lockURL, lock.LockHandle)
		}
	}()

	// Step 2: PUT the text elements (stateful — stay within the lock session, issue #88)
	params := url.Values{}
	params.Set("lockHandle", lock.LockHandle)
	if transport != "" {
		params.Set("corrNr", transport)
	}
	mediaType := fmt.Sprintf("application/vnd.sap.adt.textelements.%s.v1", category)
	opts := &RequestOptions{
		Method:      http.MethodPut,
		Query:       params,
		Body:        []byte(body),
		ContentType: mediaType + "; charset=UTF-8",
		Stateful:    true,
	}
	if lang != "" {
		opts.OverrideLanguage = lang
	}
	if _, err := c.transport.Request(ctx, lockURL, opts); err != nil {
		result.Message = fmt.Sprintf("Failed to write text elements: %v", err)
		return result, nil
	}

	// Step 3: Unlock before activation (SAP requirement)
	if err := c.UnlockObject(ctx, lockURL, lock.LockHandle); err != nil {
		result.Message = fmt.Sprintf("Failed to unlock text elements: %v", err)
		return result, nil
	}

	// Step 4: Activate the program so the new text elements take effect at runtime
	activation, err := c.Activate(ctx, programURL, programName)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to activate: %v", err)
		result.Activation = activation
		return result, nil
	}
	result.Activation = activation
	if activation.Success {
		result.Success = true
		result.Message = fmt.Sprintf("Text pool (%s) written and program activated successfully", category)
	} else {
		result.Message = "Activation failed - check activation messages"
	}

	return result, nil
}

// CreateProgramResult represents the result of creating a program.
type CreateProgramResult struct {
	Success      bool                `json:"success"`
	ProgramName  string              `json:"programName"`
	ObjectURL    string              `json:"objectUrl"`
	SyntaxErrors []SyntaxCheckResult `json:"syntaxErrors,omitempty"`
	Activation   *ActivationResult   `json:"activation,omitempty"`
	Message      string              `json:"message,omitempty"`
}

// CreateAndActivateProgram creates a new program with source code and activates it.
// Workflow: CreateObject -> Lock -> UpdateSource -> Unlock -> Activate
func (c *Client) CreateAndActivateProgram(ctx context.Context, programName string, description string, packageName string, source string, transport string) (*CreateProgramResult, error) {
	programName = strings.ToUpper(programName)
	packageName = strings.ToUpper(packageName)

	// Unified mutation policy gate (op type + package + transport)
	if err := c.checkMutation(ctx, MutationContext{
		Op:        OpWorkflow,
		OpName:    "CreateAndActivateProgram",
		Package:   packageName,
		Transport: transport,
	}); err != nil {
		return nil, err
	}

	objectURL := fmt.Sprintf("/sap/bc/adt/programs/programs/%s", url.PathEscape(programName))
	sourceURL := objectURL + "/source/main"

	result := &CreateProgramResult{
		ProgramName: programName,
		ObjectURL:   objectURL,
	}

	// Step 1: Create the program
	err := c.CreateObject(ctx, CreateObjectOptions{
		ObjectType:  ObjectTypeProgram,
		Name:        programName,
		Description: description,
		PackageName: packageName,
		Transport:   transport,
	})
	if err != nil {
		result.Message = fmt.Sprintf("Failed to create program: %v", err)
		return result, nil
	}

	// Step 2: Lock
	lock, err := c.LockObject(ctx, objectURL, "MODIFY")
	if err != nil {
		result.Message = fmt.Sprintf("Failed to lock object: %v", err)
		return result, nil
	}

	defer func() {
		if !result.Success {
			c.UnlockObject(ctx, objectURL, lock.LockHandle)
		}
	}()

	// Step 3: Update source
	err = c.UpdateSource(ctx, sourceURL, source, lock.LockHandle, transport)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to update source: %v", err)
		return result, nil
	}

	// Step 4: Unlock
	err = c.UnlockObject(ctx, objectURL, lock.LockHandle)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to unlock object: %v", err)
		return result, nil
	}

	// Step 5: Activate
	activation, err := c.Activate(ctx, objectURL, programName)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to activate: %v", err)
		result.Activation = activation
		return result, nil
	}

	result.Activation = activation
	if activation.Success {
		result.Success = true
		result.Message = "Program created and activated successfully"
	} else {
		result.Message = "Activation failed - check activation messages"
	}

	return result, nil
}

// CreateClassWithTestsResult represents the result of creating a class with unit tests.
type CreateClassWithTestsResult struct {
	Success        bool              `json:"success"`
	ClassName      string            `json:"className"`
	ObjectURL      string            `json:"objectUrl"`
	Activation     *ActivationResult `json:"activation,omitempty"`
	UnitTestResult *UnitTestResult   `json:"unitTestResult,omitempty"`
	Message        string            `json:"message,omitempty"`
}

// CreateClassWithTests creates a new class with unit tests and runs them.
// Workflow: CreateObject -> Lock -> UpdateSource -> CreateTestInclude -> UpdateClassInclude -> Unlock -> Activate -> RunUnitTests
func (c *Client) CreateClassWithTests(ctx context.Context, className string, description string, packageName string, classSource string, testSource string, transport string) (*CreateClassWithTestsResult, error) {
	className = strings.ToUpper(className)
	packageName = strings.ToUpper(packageName)

	// Unified mutation policy gate (op type + package + transport)
	if err := c.checkMutation(ctx, MutationContext{
		Op:        OpWorkflow,
		OpName:    "CreateClassWithTests",
		Package:   packageName,
		Transport: transport,
	}); err != nil {
		return nil, err
	}

	objectURL := fmt.Sprintf("/sap/bc/adt/oo/classes/%s", url.PathEscape(className))
	sourceURL := objectURL + "/source/main"

	result := &CreateClassWithTestsResult{
		ClassName: className,
		ObjectURL: objectURL,
	}

	// Step 1: Create the class
	err := c.CreateObject(ctx, CreateObjectOptions{
		ObjectType:  ObjectTypeClass,
		Name:        className,
		Description: description,
		PackageName: packageName,
		Transport:   transport,
	})
	if err != nil {
		result.Message = fmt.Sprintf("Failed to create class: %v", err)
		return result, nil
	}

	// Step 2: Lock
	lock, err := c.LockObject(ctx, objectURL, "MODIFY")
	if err != nil {
		result.Message = fmt.Sprintf("Failed to lock object: %v", err)
		return result, nil
	}

	defer func() {
		if !result.Success {
			c.UnlockObject(ctx, objectURL, lock.LockHandle)
		}
	}()

	// Step 3: Update main source
	err = c.UpdateSource(ctx, sourceURL, classSource, lock.LockHandle, transport)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to update class source: %v", err)
		return result, nil
	}

	// Step 4: Create test include
	err = c.CreateTestInclude(ctx, className, lock.LockHandle, transport)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to create test include: %v", err)
		return result, nil
	}

	// Step 5: Update test include
	err = c.UpdateClassInclude(ctx, className, ClassIncludeTestClasses, testSource, lock.LockHandle, transport)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to update test source: %v", err)
		return result, nil
	}

	// Step 6: Unlock
	err = c.UnlockObject(ctx, objectURL, lock.LockHandle)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to unlock object: %v", err)
		return result, nil
	}

	// Step 7: Activate
	activation, err := c.Activate(ctx, objectURL, className)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to activate: %v", err)
		result.Activation = activation
		return result, nil
	}
	result.Activation = activation

	if !activation.Success {
		result.Message = "Activation failed - check activation messages"
		return result, nil
	}

	// Step 8: Run unit tests
	flags := DefaultUnitTestFlags()
	testResult, err := c.RunUnitTests(ctx, objectURL, &flags)
	if err != nil {
		result.Message = fmt.Sprintf("Class activated but unit tests failed to run: %v", err)
		result.Success = true // Class was created successfully
		return result, nil
	}

	result.UnitTestResult = testResult
	result.Success = true
	result.Message = "Class created, activated, and unit tests executed successfully"

	return result, nil
}
