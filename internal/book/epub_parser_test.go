package book

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractEPUB(t *testing.T) {
	epubPath := writeTestEPUB(t, map[string]string{
		"META-INF/container.xml": `<?xml version="1.0"?><container><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`,
		"OEBPS/content.opf": `<package><manifest>
			<item id="one" href="one.xhtml" media-type="application/xhtml+xml"/>
			<item id="two" href="two.xhtml" media-type="application/xhtml+xml"/>
			<item id="css" href="book.css" media-type="text/css"/>
		</manifest><spine><itemref idref="one"/><itemref idref="css"/><itemref idref="two"/></spine></package>`,
		"OEBPS/one.xhtml": `<html><head><title>Chapter &amp; One</title><style>.hidden { display: none; }</style></head><body><h1>Chapter &amp; One</h1><p>First paragraph.</p><script>ignored()</script><p>Second paragraph.</p></body></html>`,
		"OEBPS/two.xhtml": `<html><body><h2>Second chapter</h2><p>Final text.</p></body></html>`,
	})
	outputDir := filepath.Join(t.TempDir(), "chapters")

	require.NoError(t, ExtractEPUB(epubPath, outputDir))

	first, err := os.ReadFile(filepath.Join(outputDir, "001_Chapter_One.txt"))
	require.NoError(t, err)
	require.Equal(t, "Chapter & One\n\nChapter & OneChapter & One\nFirst paragraph.\nSecond paragraph.", string(first))
	second, err := os.ReadFile(filepath.Join(outputDir, "002_Second_chapter.txt"))
	require.NoError(t, err)
	require.Equal(t, "Second chapter\n\nSecond chapter\nFinal text.", string(second))
}

func TestExtractEPUBReturnsErrorWhenNoChapterCanBeExtracted(t *testing.T) {
	epubPath := writeTestEPUB(t, map[string]string{
		"META-INF/container.xml": `<container><rootfiles><rootfile full-path="content.opf"/></rootfiles></container>`,
		"content.opf":            `<package><manifest><item id="empty" href="empty.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="empty"/></spine></package>`,
		"empty.xhtml":            `<html><body><script>ignored()</script></body></html>`,
	})

	err := ExtractEPUB(epubPath, filepath.Join(t.TempDir(), "chapters"))
	require.ErrorContains(t, err, "no chapter content extracted")
}

func writeTestEPUB(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "book.epub")
	output, err := os.Create(path)
	require.NoError(t, err)
	writer := zip.NewWriter(output)
	for name, content := range files {
		entry, err := writer.Create(name)
		require.NoError(t, err)
		_, err = entry.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	require.NoError(t, output.Close())
	return path
}
