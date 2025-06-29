package main

import (
	"bufio"
	"strings"
	"text/template"
)

type Redirect struct {
	From string
	To   string
}

type TemplateData struct {
	Redirects []Redirect
}

// GenerateRedirectVCL reads a redirect map, processes a VCL template, and writes to the output path
func GenerateRedirectVCL(mapFileContent, templateContent string) (string, error) {

	// Parse redirect map
	var redirects []Redirect
	scanner := bufio.NewScanner(strings.NewReader(mapFileContent))
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) == 2 {
			redirects = append(redirects, Redirect{
				From: parts[0],
				To:   parts[1],
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	// Parse template
	tmpl, err := template.New("vcl").Parse(templateContent)
	if err != nil {
		return "", err
	}

	// Render to string
	var buf strings.Builder
	if err := tmpl.Execute(&buf, TemplateData{Redirects: redirects}); err != nil {
		return "", err
	}

	return buf.String(), nil
}
