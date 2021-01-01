// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"text/template"

	"github.com/pkg/errors"
	"github.com/wedaly/aretext/internal/pkg/syntax/parser"
	"github.com/wedaly/aretext/internal/pkg/syntax/rules"
)

func main() {
	generateTokenizer("JsonTokenizer", rules.JsonRules, "json_tokenizer.go")
}

func generateTokenizer(tokenizerName string, tokenizerRules []parser.TokenizerRule, outputPath string) {
	fmt.Printf("Generating tokenizer %s to %s\n", tokenizerName, outputPath)

	tokenizer, err := parser.GenerateTokenizer(tokenizerRules)
	if err != nil {
		log.Fatalf("Error generating tokenizer %s: %v\n", tokenizerName, err)
	}

	if err := writeTokenizer(tokenizer, tokenizerName, outputPath); err != nil {
		log.Fatalf("Error writing tokenizer %s to %s: %v\n", tokenizerName, outputPath, err)
	}
}

func writeTokenizer(tokenizer *parser.Tokenizer, tokenizerName string, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return errors.Wrapf(err, "os.Create")
	}
	defer f.Close()

	tmplStr := `// This file is generated by gen_tokenizers.go.  DO NOT EDIT.
package syntax

import "github.com/wedaly/aretext/internal/pkg/syntax/parser"

var {{ .TokenizerName }} *parser.Tokenizer

func init() {
	{{ .TokenizerName }} = &parser.Tokenizer{
		StateMachine: &parser.Dfa{
			NumStates: {{ .Tokenizer.StateMachine.NumStates }},
			StartState: {{ .Tokenizer.StateMachine.StartState }},
			Transitions: {{ printf "%#v" .Tokenizer.StateMachine.Transitions }},
			AcceptActions: {{ printf "%#v" .Tokenizer.StateMachine.AcceptActions }},
		},
		Rules: []parser.TokenizerRule{
			{{ range $rule := .Tokenizer.Rules }}
			{
				Regexp: {{ printf "%q" $rule.Regexp }},
				TokenRole: {{ $rule.TokenRole }},
			},
			{{ end }}
		},
	}
}
`

	tmpl, err := template.New("tokenizer").Parse(tmplStr)
	if err != nil {
		return errors.Wrapf(err, "template.New")
	}

	return tmpl.Execute(f, map[string]interface{}{
		"TokenizerName": tokenizerName,
		"Tokenizer":     tokenizer,
	})
}
