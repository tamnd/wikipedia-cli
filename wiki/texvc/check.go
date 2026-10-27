package texvc

import (
	"fmt"
	"strings"
)

// open is a group that Check has seen open.
type open struct {
	kind string // "{", "left" or the environment name
	env  bool
}

// Check runs a small balance check over tex: braces, \begin and \end, and
// \left and \right must pair up and nest. It returns one message for each
// problem, or nil. It is not a TeX parser; the KaTeX parse happens later.
func Check(tex string) []string {
	var stack []open
	var errs []string
	for i := 0; i < len(tex); i++ {
		c := tex[i]
		switch c {
		case '%':
			for i < len(tex) && tex[i] != '\n' {
				i++
			}
		case '{':
			stack = append(stack, open{kind: "{"})
		case '}':
			k := len(stack) - 1
			for k >= 0 && (stack[k].kind != "{" || stack[k].env) {
				k--
			}
			if k < 0 {
				errs = append(errs, "unbalanced braces: } without {")
				continue
			}
			errs = append(errs, unclosed(stack[k+1:])...)
			stack = stack[:k]
		case '\\':
			j := i + 1
			for j < len(tex) && isASCIILetter(tex[j]) {
				j++
			}
			if j == i+1 {
				i++ // control symbol such as \{ or \\
				continue
			}
			name := tex[i+1 : j]
			i = j - 1
			switch name {
			case "left":
				stack = append(stack, open{kind: "left"})
			case "right":
				if len(stack) == 0 || stack[len(stack)-1].kind != "left" || stack[len(stack)-1].env {
					errs = append(errs, `\right without \left`)
					continue
				}
				stack = stack[:len(stack)-1]
			case "begin", "end":
				env, n := envName(tex[j:])
				if n == 0 {
					errs = append(errs, `\`+name+" without an environment name")
					continue
				}
				i = j + n - 1
				if name == "begin" {
					stack = append(stack, open{kind: env, env: true})
					continue
				}
				k := len(stack) - 1
				for k >= 0 && (!stack[k].env || stack[k].kind != env) {
					k--
				}
				if k >= 0 {
					errs = append(errs, unclosed(stack[k+1:])...)
					stack = stack[:k]
					continue
				}
				if n := len(stack); n > 0 && stack[n-1].env {
					errs = append(errs, fmt.Sprintf(`\end{%s} closes \begin{%s}`, env, stack[n-1].kind))
					stack = stack[:n-1]
					continue
				}
				errs = append(errs, fmt.Sprintf(`\end{%s} without \begin`, env))
			}
		}
	}
	return append(errs, unclosed(stack)...)
}

// unclosed reports the openers in stack that were never closed, innermost
// first.
func unclosed(stack []open) []string {
	var errs []string
	for k := len(stack) - 1; k >= 0; k-- {
		switch o := stack[k]; {
		case o.env:
			errs = append(errs, fmt.Sprintf(`\begin{%s} without \end`, o.kind))
		case o.kind == "left":
			errs = append(errs, `\left without \right`)
		default:
			errs = append(errs, "unbalanced braces: { without }")
		}
	}
	return errs
}

// envName reads "{name}" at the start of s, after optional spaces, and returns
// the name and the bytes it used.
func envName(s string) (string, int) {
	i := 0
	for i < len(s) && s[i] == ' ' {
		i++
	}
	if i >= len(s) || s[i] != '{' {
		return "", 0
	}
	end := strings.IndexByte(s[i:], '}')
	if end < 0 {
		return "", 0
	}
	name := strings.TrimSpace(s[i+1 : i+end])
	if name == "" {
		return "", 0
	}
	return name, i + end + 1
}
