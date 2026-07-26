package book

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type epubContainer struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type epubManifestItem struct {
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

type epubPackage struct {
	Manifest struct {
		Items []epubManifestItem `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		Items []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

// ExtractEPUB extracts each non-empty HTML document in an EPUB spine into a
// numbered plain-text chapter in outputDir.
func ExtractEPUB(epubPath, outputDir string) error {
	zr, err := zip.OpenReader(epubPath)
	if err != nil {
		return fmt.Errorf("opening epub: %w", err)
	}
	defer zr.Close()

	files := make(map[string]*zip.File, len(zr.File))
	for _, file := range zr.File {
		files[file.Name] = file
	}

	containerBytes, err := readEPUBFile(files, "META-INF/container.xml")
	if err != nil {
		return fmt.Errorf("reading container.xml: %w", err)
	}
	var container epubContainer
	if err := xml.Unmarshal(containerBytes, &container); err != nil {
		return fmt.Errorf("parsing container.xml: %w", err)
	}
	if len(container.Rootfiles) == 0 {
		return fmt.Errorf("no rootfile found in container.xml")
	}

	opfPath := container.Rootfiles[0].FullPath
	opfBytes, err := readEPUBFile(files, opfPath)
	if err != nil {
		return fmt.Errorf("reading opf: %w", err)
	}
	var pkg epubPackage
	if err := xml.Unmarshal(opfBytes, &pkg); err != nil {
		return fmt.Errorf("parsing opf: %w", err)
	}
	if len(pkg.Spine.Items) == 0 {
		return fmt.Errorf("no spine items found — is this a valid EPUB?")
	}

	manifestByID := make(map[string]epubManifestItem, len(pkg.Manifest.Items))
	for _, item := range pkg.Manifest.Items {
		manifestByID[item.ID] = item
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	chapters := 0
	opfDir := path.Dir(opfPath)
	for _, spineItem := range pkg.Spine.Items {
		item, ok := manifestByID[spineItem.IDRef]
		if !ok || !isEPUBHTML(item.MediaType, item.Href) {
			continue
		}

		raw, err := readEPUBFile(files, path.Join(opfDir, item.Href))
		if err != nil {
			continue
		}
		text, title := epubHTMLToText(string(raw))
		if text == "" {
			continue
		}

		chapters++
		filename := fmt.Sprintf("%03d_%s.txt", chapters, epubSlug(title, item.Href))
		content := text
		if title != "" {
			content = title + "\n\n" + text
		}
		if err := os.WriteFile(filepath.Join(outputDir, filename), []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
	}
	if chapters == 0 {
		return fmt.Errorf("no chapter content extracted — check the EPUB structure")
	}
	return nil
}

func isEPUBHTML(mediaType, href string) bool {
	return strings.Contains(strings.ToLower(mediaType), "html") ||
		strings.EqualFold(path.Ext(href), ".html") ||
		strings.EqualFold(path.Ext(href), ".xhtml") ||
		strings.EqualFold(path.Ext(href), ".htm")
}

func readEPUBFile(files map[string]*zip.File, name string) ([]byte, error) {
	cleanName := strings.TrimPrefix(name, "/")
	file, ok := files[name]
	if !ok {
		for filename, candidate := range files {
			if strings.TrimPrefix(filename, "/") == cleanName {
				file, ok = candidate, true
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("file not found in epub: %s", name)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

var (
	epubScriptStyle = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	epubTitleTag    = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	epubHeading     = regexp.MustCompile(`(?is)<h[1-6][^>]*>(.*?)</h[1-6]>`)
	epubBlockClose  = regexp.MustCompile(`(?is)</(p|div|br|h[1-6]|li|tr)\s*/?>`)
	epubTag         = regexp.MustCompile(`(?is)<[^>]+>`)
	epubMultiBlank  = regexp.MustCompile(`\n{3,}`)
	epubMultiSpace  = regexp.MustCompile(`[ \t]{2,}`)
	epubSlugInvalid = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

func epubHTMLToText(doc string) (text, title string) {
	if match := epubTitleTag.FindStringSubmatch(doc); match != nil {
		title = epubCleanInline(match[1])
	}
	if title == "" {
		if match := epubHeading.FindStringSubmatch(doc); match != nil {
			title = epubCleanInline(match[1])
		}
	}

	body := epubScriptStyle.ReplaceAllString(doc, "")
	body = epubBlockClose.ReplaceAllString(body, "\n")
	body = html.UnescapeString(epubTag.ReplaceAllString(body, ""))
	body = epubMultiSpace.ReplaceAllString(body, " ")
	lines := strings.Split(body, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	body = epubMultiBlank.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
	return strings.TrimSpace(body), title
}

func epubCleanInline(value string) string {
	return strings.TrimSpace(html.UnescapeString(epubTag.ReplaceAllString(value, "")))
}

func epubSlug(title, fallback string) string {
	base := title
	if base == "" {
		base = strings.TrimSuffix(path.Base(fallback), path.Ext(fallback))
	}
	slug := strings.Trim(epubSlugInvalid.ReplaceAllString(base, "_"), "_")
	if slug == "" {
		slug = "chapter"
	}
	if len(slug) > 60 {
		slug = slug[:60]
	}
	return slug
}
