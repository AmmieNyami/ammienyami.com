package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"
)

var ErrTemplateUnexpectedToken = errors.New("unexpected token")
var ErrTemplateUnexpectedEof = errors.New("unexpected end of file")
var ErrTemplateVariableNotFound = errors.New("variable not present in template input")
var ErrTemplateIncorrectArgsCount = errors.New("provided an incorrect number of arguments")
var ErrTemplateUnterminatedCodeBlock = errors.New("unterminated code block")
var ErrTemplateInternalFunctionExecutionError = errors.New("internal error while executing function")

type Location struct {
	path   string
	row    int
	column int
}

type TemplateContext struct {
	StaticDir string
	Content   string
	Variables map[string]string
}

type TemplatePortion interface {
	Render() (string, error)
}

type TemplateTextPortion struct {
	text string
}

func (templatePortion TemplateTextPortion) Render() (string, error) {
	return templatePortion.text, nil
}

func errUnexpectedToken(tok *Token) error {
	return fmt.Errorf(
		"%s:%d:%d: ERROR: failed to render template: %q: %w",
		tok.location.path, tok.location.row, tok.location.column, tok.value,
		ErrTemplateUnexpectedToken,
	)
}

func errUnexpectedEof(loc Location) error {
	return fmt.Errorf(
		"%s:%d:%d: ERROR: failed to render template: %w",
		loc.path, loc.row, loc.column,
		ErrTemplateUnexpectedEof,
	)
}

func errVariableNotFound(tok *Token) error {
	return fmt.Errorf(
		"%s:%d:%d: ERROR: failed to render template: %q: %w",
		tok.location.path, tok.location.row, tok.location.column, tok.value,
		ErrTemplateVariableNotFound,
	)
}

func errIncorrectArgsCount(tok *Token) error {
	if tok.kind != TokenKindWord {
		panic("`errIncorrectArgsCount` got an unexpected token kind as an argument")
	}
	return fmt.Errorf(
		"%s:%d:%d: ERROR: failed to render template: %q: %w",
		tok.location.path, tok.location.row, tok.location.column, tok.value,
		ErrTemplateIncorrectArgsCount,
	)
}

func errTemplateInternalFunctionExecutionError(tok *Token) error {
	return fmt.Errorf(
		"%s:%d:%d: ERROR: failed to render template: %q: %w",
		tok.location.path, tok.location.row, tok.location.column, tok.value,
		ErrTemplateInternalFunctionExecutionError,
	)
}

func mustNextToken(tokenizer *Tokenizer, loc Location) (*Token, error) {
	tok, err := tokenizer.NextToken()
	if err != nil {
		return nil, err
	}
	if tok == nil {
		return nil, errUnexpectedEof(loc)
	}
	return tok, nil
}

func parseFunctionArgs(tokenizer *Tokenizer, loc Location) ([]Token, error) {
	tok, err := mustNextToken(tokenizer, loc)
	if err != nil {
		return nil, err
	}
	if !tok.IsSingleChar('(') {
		return nil, errUnexpectedToken(tok)
	}

	args := []Token{}

	tok, err = mustNextToken(tokenizer, loc)
	if err != nil {
		return nil, err
	}
	if tok.IsSingleChar(')') {
		return args, nil
	}
	if tok.IsSingleChar(',') {
		return nil, errUnexpectedToken(tok)
	}
	args = append(args, *tok)

	for {
		tok, err = mustNextToken(tokenizer, loc)
		if err != nil {
			return nil, err
		}
		if tok.IsSingleChar(')') {
			return args, nil
		}
		if !tok.IsSingleChar(',') {
			return nil, errUnexpectedToken(tok)
		}

		tok, err = mustNextToken(tokenizer, loc)
		if err != nil {
			return nil, err
		}
		if tok.IsSingleChar(')') || tok.IsSingleChar(',') {
			return nil, errUnexpectedToken(tok)
		}
		args = append(args, *tok)
	}
}

type TemplateCodePortion struct {
	ctx      TemplateContext
	location Location
	code     string
}

func (templatePortion TemplateCodePortion) renderVar(tokenizer *Tokenizer, funcTok *Token) (string, error) {
	args, err := parseFunctionArgs(tokenizer, templatePortion.location)
	if err != nil {
		return "", err
	}
	if len(args) != 1 {
		return "", errIncorrectArgsCount(funcTok)
	}

	varNameToken := args[0]
	if varNameToken.kind != TokenKindString {
		return "", errUnexpectedToken(&varNameToken)
	}

	varName := varNameToken.value
	value, ok := templatePortion.ctx.Variables[varName]
	if !ok {
		return "", errVariableNotFound(&varNameToken)
	}
	return value, nil
}

func (templatePortion TemplateCodePortion) renderContent(tokenizer *Tokenizer, funcTok *Token) (string, error) {
	args, err := parseFunctionArgs(tokenizer, templatePortion.location)
	if err != nil {
		return "", err
	}
	if len(args) != 0 {
		return "", errIncorrectArgsCount(funcTok)
	}

	return templatePortion.ctx.Content, nil
}

func (templatePortion TemplateCodePortion) renderChooseRandomTopLevelFileFromStaticPath(tokenizer *Tokenizer, funcTok *Token) (string, error) {
	args, err := parseFunctionArgs(tokenizer, templatePortion.location)
	if err != nil {
		return "", err
	}
	if len(args) < 1 || len(args) > 2 {
		return "", errIncorrectArgsCount(funcTok)
	}

	dirPathToken := args[0]
	if dirPathToken.kind != TokenKindString {
		return "", errUnexpectedToken(&dirPathToken)
	}
	dirPath := dirPathToken.value

	// The second parameter is optional and specifies
	// a file extension to *ignore*.
	ext := ""
	if len(args) > 1 {
		extToken := args[1]
		if extToken.kind != TokenKindString {
			return "", errUnexpectedToken(&extToken)
		}
		ext = extToken.value
	}

	dirEntries, err := os.ReadDir(path.Join(templatePortion.ctx.StaticDir, path.Clean(dirPath)))
	if err != nil {
		return "", errTemplateInternalFunctionExecutionError(funcTok)
	}

	var files []string
	for _, e := range dirEntries {
		if ext != "" && path.Ext(e.Name()) == ext {
			continue
		}
		files = append(files, e.Name())
	}

	if len(files) < 1 {
		return "", errTemplateInternalFunctionExecutionError(funcTok)
	}

	chosenIndex := rand.Intn(len(files))
	randomFile := files[chosenIndex]

	return path.Join("/", templatePortion.ctx.StaticDir, dirPath, randomFile), nil
}

func (templatePortion TemplateCodePortion) Render() (string, error) {
	tokenizer := NewTokenizer(templatePortion.code, templatePortion.location)
	loc := templatePortion.location

	tok, err := tokenizer.NextToken()
	if err != nil {
		return "", err
	}
	if tok == nil {
		return "", errUnexpectedEof(loc)
	}

	switch {
	case tok.IsWord() && tok.value == "var":
		return templatePortion.renderVar(tokenizer, tok)

	case tok.IsWord() && tok.value == "content":
		return templatePortion.renderContent(tokenizer, tok)

	case tok.IsWord() && tok.value == "chooseRandomTopLevelFileFromStaticPath":
		return templatePortion.renderChooseRandomTopLevelFileFromStaticPath(tokenizer, tok)

	default:
		return "", errUnexpectedToken(tok)
	}
}

type Template struct {
	portions []TemplatePortion
}

func NewTemplateFromString(initialLocation Location, textContent string, codeCtx TemplateContext) (Template, error) {
	portions := []TemplatePortion{}
	runes := []rune(textContent)

	i := 0
	location := initialLocation
	accumulatedText := ""
	codeMode := false
	codeModeLocation := Location{}

	advance := func() rune {
		char := runes[i]
		i++
		if char == '\n' {
			location.row++
			location.column = 1
		} else {
			location.column++
		}
		return char
	}

	for i < len(runes) {
		char := runes[i]

		if char == '\\' {
			advance()
			if i >= len(runes) {
				accumulatedText += string(char)
				break
			}
			accumulatedText += string(advance())
			continue
		}

		if char == '{' && !codeMode {
			portions = append(portions, TemplateTextPortion{text: accumulatedText})
			accumulatedText = ""
			codeMode = true
			advance()
			codeModeLocation = location
			continue
		}

		if char == '}' && codeMode {
			codeMode = false
			portions = append(portions, TemplateCodePortion{
				ctx:      codeCtx,
				location: codeModeLocation,
				code:     accumulatedText,
			})
			accumulatedText = ""
			advance()
			continue
		}

		accumulatedText += string(advance())
	}

	if codeMode {
		return Template{}, fmt.Errorf(
			"%s:%d:%d: ERROR: failed to parse template file: %w",
			codeModeLocation.path, codeModeLocation.row, codeModeLocation.column-1,
			ErrTemplateUnterminatedCodeBlock,
		)
	}

	if accumulatedText != "" {
		portions = append(portions, TemplateTextPortion{text: accumulatedText})
	}

	return Template{portions}, nil
}

func NewTemplateFromFile(path string, codeCtx TemplateContext) (Template, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return Template{}, fmt.Errorf("%s: ERROR: failed to read template file: %w", path, err)
	}
	return NewTemplateFromString(Location{path, 1, 1}, string(contentBytes), codeCtx)
}

func (t *Template) Render() (string, error) {
	var builder strings.Builder

	for _, portion := range t.portions {
		renderedPortion, err := portion.Render()
		if err != nil {
			return "", err
		}
		builder.WriteString(renderedPortion)
	}

	return builder.String(), nil
}

type TemplateInput struct {
	variables map[string]string
	content   *Template
}

func NewTemplateInputFromString(initialLocation Location, textContent string, staticDir string) (TemplateInput, error) {
	runes := []rune(textContent)

	i := 0
	location := initialLocation

	advance := func() rune {
		char := runes[i]
		i++
		if char == '\n' {
			location.row++
			location.column = 1
		} else {
			location.column++
		}
		return char
	}

	var jsonHeader strings.Builder
	bracketBalance := 0
	inString := false
	escaped := false

	for i < len(runes) {
		r := runes[i]

		if escaped {
			escaped = false
		} else if inString {
			switch r {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
		} else {
			switch r {
			case '"':
				inString = true
			case '{':
				bracketBalance++
			case '}':
				bracketBalance--
			}
		}

		jsonHeader.WriteRune(r)
		advance()

		if !inString && bracketBalance == 0 {
			break
		}
	}

	templateInputVariables := make(map[string]string)

	err := UnmarshalJsonWithComments(jsonHeader.String(), &templateInputVariables)
	if err != nil {
		return TemplateInput{}, fmt.Errorf(
			"%s: ERROR: failed to parse template input file: invalid JSON header",
			initialLocation.path,
		)
	}

	var templateInputContent strings.Builder
	templateInputContentLocation := location

	for i < len(runes) {
		templateInputContent.WriteRune(runes[i])
		advance()
	}

	templateInputContentTemplate, err := NewTemplateFromString(
		templateInputContentLocation,
		templateInputContent.String(),
		TemplateContext{
			StaticDir: staticDir,
			Content:   "[content()]",
			Variables: templateInputVariables,
		},
	)
	if err != nil {
		return TemplateInput{}, err
	}

	return TemplateInput{
		variables: templateInputVariables,
		content:   &templateInputContentTemplate,
	}, nil
}

func NewTemplateInputFromFile(path string, staticDir string) (TemplateInput, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return TemplateInput{}, fmt.Errorf("%s: ERROR: failed to read template input file: %w", path, err)
	}
	return NewTemplateInputFromString(Location{path, 1, 1}, string(contentBytes), staticDir)
}

func (templateInput TemplateInput) Render() (string, error) {
	return templateInput.content.Render()
}
