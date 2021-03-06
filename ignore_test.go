// Implement tests for the `ignore` library
package ignore

import (
	"os"

	"io/ioutil"
	"path/filepath"

	"fmt"
	"testing"

	"runtime"

	"github.com/stretchr/testify/assert"
)

const (
	TEST_DIR = "test_fixtures"
)

////////////////////////////////////////////////////////////

// Helper function to setup a test fixture dir and write to
// it a file with the name "fname" and content "content"
func writeFileToTestDir(fname, content string) string {
	testDirPath := "." + string(filepath.Separator) + TEST_DIR
	testFilePath := testDirPath + string(filepath.Separator) + fname
	_ = os.MkdirAll(testDirPath, 0755)
	_ = ioutil.WriteFile(testFilePath, []byte(content), os.ModePerm)
	return testFilePath
}

func cleanupTestDir() {
	_ = os.RemoveAll(fmt.Sprintf(".%s%s", string(filepath.Separator), TEST_DIR))
}

////////////////////////////////////////////////////////////

// Validate "CompileIgnoreLines()"
func TestCompileIgnoreLines(test *testing.T) {
	lines := []string{"abc/def", "a/b/c", "b"}
	object := CompileIgnoreLines(lines...)

	// MatchesPath
	// Paths which are targeted by the above "lines"
	assert.True(test, object.MatchesPath("abc/def/child"), "abc/def/child should match")
	assert.True(test, object.MatchesPath("a/b/c/d"), "a/b/c/d should match")

	// Paths which are not targeted by the above "lines"
	assert.False(test, object.MatchesPath("abc"), "abc should not match")
	assert.False(test, object.MatchesPath("def"), "def should not match")
	assert.False(test, object.MatchesPath("bd"), "bd should not match")

	object = CompileIgnoreLines("abc/def", "a/b/c", "b")

	// Paths which are targeted by the above "lines"
	assert.True(test, object.MatchesPath("abc/def/child"), "abc/def/child should match")
	assert.True(test, object.MatchesPath("a/b/c/d"), "a/b/c/d should match")

	// Paths which are not targeted by the above "lines"
	assert.False(test, object.MatchesPath("abc"), "abc should not match")
	assert.False(test, object.MatchesPath("def"), "def should not match")
	assert.False(test, object.MatchesPath("bd"), "bd should not match")
}

func TestCompileIgnoreLinesAddPattersFromLines(test *testing.T) {
	object := CompileIgnoreLines("abc/def", "a/b/c", "b").AddPatternsFromLines("efg/hij")

	// Paths which are targeted by the above "lines"
	assert.True(test, object.MatchesPath("abc/def/child"), "abc/def/child should match")
	assert.True(test, object.MatchesPath("efg/hij/child"), "efg/hij/child should match")
	assert.True(test, object.MatchesPath("a/b/c/d"), "a/b/c/d should match")

	// Paths which are not targeted by the above "lines"
	assert.False(test, object.MatchesPath("abc"), "abc should not match")
	assert.False(test, object.MatchesPath("def"), "def should not match")
	assert.False(test, object.MatchesPath("bd"), "bd should not match")
	assert.False(test, object.MatchesPath("efg"), "efg should not match")
}

func TestCompileIgnoreLinesAddPattersFromFiles(test *testing.T) {
	filename := writeFileToTestDir("test.gitignore", `
efg/hij
`)
	defer cleanupTestDir()

	object := CompileIgnoreLines("abc/def", "a/b/c", "b").AddPatternsFromFiles(filename)

	// Paths which are targeted by the above "lines"
	assert.True(test, object.MatchesPath("abc/def/child"), "abc/def/child should match")
	assert.True(test, object.MatchesPath("efg/hij/child"), "efg/hij/child should match")
	assert.True(test, object.MatchesPath("a/b/c/d"), "a/b/c/d should match")

	// Paths which are not targeted by the above "lines"
	assert.False(test, object.MatchesPath("abc"), "abc should not match")
	assert.False(test, object.MatchesPath("def"), "def should not match")
	assert.False(test, object.MatchesPath("bd"), "bd should not match")
	assert.False(test, object.MatchesPath("efg"), "efg should not match")
}

// Validate the invalid files
func TestCompileIgnoreFileInvalidFile(test *testing.T) {
	object, err := CompileIgnoreFile("./test_fixtures/invalid.file")
	assert.Nil(test, object, "object should be nil")
	assert.NotNil(test, err, "error should be unknown file / dir")
}

// Validate the an empty files
func TestCompileIgnoreLinesEmptyFile(test *testing.T) {
	filename := writeFileToTestDir("test.gitignore", ``)
	defer cleanupTestDir()

	object, err := CompileIgnoreFile(filename)
	assert.NoError(test, err)
	assert.NotNil(test, object, "object should not be nil")

	assert.False(test, object.MatchesPath("a"), "should not match any path")
	assert.False(test, object.MatchesPath("a/b"), "should not match any path")
	assert.False(test, object.MatchesPath(".foobar"), "should not match any path")
}

// Validate the correct handling of the negation operator "!"
func TestCompileIgnoreLinesHandleIncludePattern(test *testing.T) {
	filename := writeFileToTestDir("test.gitignore", `
# exclude everything except directory foo/bar
/*
!/foo
/foo/*
!/foo/bar
`)
	defer cleanupTestDir()

	object, err := CompileIgnoreFile(filename)
	assert.NoError(test, err)
	assert.NotNil(test, object, "object should not be nil")

	assert.True(test, object.MatchesPath("a"), "a should match")
	assert.True(test, object.MatchesPath("foo/baz"), "foo/baz should match")
	assert.False(test, object.MatchesPath("foo"), "foo should not match")
	assert.False(test, object.MatchesPath("/foo/bar"), "/foo/bar should not match")
}

// Validate the correct handling of comments and empty lines
func TestCompileIgnoreLinesHandleSpaces(test *testing.T) {
	filename := writeFileToTestDir("test.gitignore", `
#
# A comment

# Another comment


    # Invalid Comment

abc/def

\!shouldmatch
!shouldnotmatch
`)
	defer cleanupTestDir()

	object, err := CompileIgnoreFile(filename)
	assert.NoError(test, err)
	assert.NotNil(test, object, "object should not be nil")

	assert.Equal(test, 4, len(object.patterns), "should have two regex pattern")
	assert.False(test, object.MatchesPath("abc/abc"), "/abc/abc should not match")
	assert.True(test, object.MatchesPath("abc/def"), "/abc/def should match")
	assert.True(test, object.MatchesPath(`!shouldmatch`), `'!shouldmatch' should match`)
	assert.False(test, object.MatchesPath(`shouldnotmatch`), `'shouldnotmatc' should not match`)
}

// Validate the correct handling of leading / chars
func TestCompileIgnoreLinesHandleLeadingSlash(test *testing.T) {
	filename := writeFileToTestDir("test.gitignore", `
/a/b/c
d/e/f
/g
`)
	defer cleanupTestDir()

	object, err := CompileIgnoreFile(filename)
	assert.NoError(test, err)
	assert.NotNil(test, object, "object should not be nil")

	assert.Equal(test, 3, len(object.patterns), "should have 3 regex patterns")
	assert.True(test, object.MatchesPath("a/b/c"), "a/b/c should match")
	assert.True(test, object.MatchesPath("a/b/c/d"), "a/b/c/d should match")
	assert.True(test, object.MatchesPath("d/e/f"), "d/e/f should match")
	assert.True(test, object.MatchesPath("g"), "g should match")
}

// Validate the correct handling of files starting with # or !
func TestCompileIgnoreLinesHandleLeadingSpecialChars(test *testing.T) {
	filename := writeFileToTestDir("test.gitignore", `
# Comment
\#file.txt
\!file.txt
file.txt
!otherfile.txt
`)
	defer cleanupTestDir()

	object, err := CompileIgnoreFile(filename)
	assert.NoError(test, err)
	assert.NotNil(test, object, "object should not be nil")

	assert.True(test, object.MatchesPath("#file.txt"), "#file.txt should match")
	assert.True(test, object.MatchesPath("!file.txt"), "!file.txt should match")
	assert.True(test, object.MatchesPath("a/!file.txt"), "a/!file.txt should match")
	assert.False(test, object.MatchesPath("otherfile.txt"), "otherfile.txt should not match")
	assert.False(test, object.MatchesPath("a/otherfile.txt"), "a/otherfile.txt should not match")
	assert.True(test, object.MatchesPath("file.txt"), "file.txt should match")
	assert.True(test, object.MatchesPath("a/file.txt"), "a/file.txt should match")
	assert.False(test, object.MatchesPath("file2.txt"), "file2.txt should not match")

}

// Validate the correct handling matching files only within a given folder
func TestCompileIgnoreLinesHandleAllFilesInDir(test *testing.T) {
	filename := writeFileToTestDir("test.gitignore", `
Documentation/*.html
`)
	defer cleanupTestDir()

	object, err := CompileIgnoreFile(filename)
	assert.NoError(test, err)
	assert.NotNil(test, object, "object should not be nil")

	assert.True(test, object.MatchesPath("Documentation/git.html"), "Documentation/git.html should match")
	assert.False(test, object.MatchesPath("Documentation/ppc/ppc.html"), "Documentation/ppc/ppc.html should not match")
	assert.False(test, object.MatchesPath("tools/perf/Documentation/perf.html"), "tools/perf/Documentation/perf.html should not match")
}

// Validate the correct handling of "**"
func TestCompileIgnoreLinesHandleDoubleStar(test *testing.T) {
	filename := writeFileToTestDir("test.gitignore", `
**/foo
bar
`)
	defer cleanupTestDir()

	object, err := CompileIgnoreFile(filename)
	assert.NoError(test, err)
	assert.NotNil(test, object, "object should not be nil")

	assert.True(test, object.MatchesPath("foo"), "foo should match")
	assert.True(test, object.MatchesPath("baz/foo"), "baz/foo should match")
	assert.True(test, object.MatchesPath("bar"), "bar should match")
	assert.True(test, object.MatchesPath("baz/bar"), "baz/bar should match")
}

// Validate the correct handling of leading slash
func TestCompileIgnoreLinesHandleLeadingSlashPath(test *testing.T) {
	filename := writeFileToTestDir("test.gitignore", `
/*.c
`)
	defer cleanupTestDir()

	object, err := CompileIgnoreFile(filename)
	assert.NoError(test, err)
	assert.NotNil(test, object, "object should not be nil")

	assert.True(test, object.MatchesPath("hello.c"), "hello.c should match")
	assert.False(test, object.MatchesPath("foo/hello.c"), "foo/hello.c should not match")
}

func ExampleCompileIgnoreLines() {
	ignoreObject := CompileIgnoreLines([]string{"node_modules", "*.out", "foo/*.c"}...)

	// You can test the ignoreObject against various paths using the
	// "MatchesPath()" interface method. This pretty much is up to
	// the users interpretation. In the case of a ".gitignore" file,
	// a "match" would indicate that a given path would be ignored.
	fmt.Println(ignoreObject.MatchesPath("node_modules/test/foo.js"))
	fmt.Println(ignoreObject.MatchesPath("node_modules2/test.out"))
	fmt.Println(ignoreObject.MatchesPath("test/foo.js"))

	// Output:
	// true
	// true
	// false
}

func TestCompileIgnoreLinesCheckNestedDotFiles(test *testing.T) {
	lines := []string{
		"**/external/**/*.md",
		"**/external/**/*.json",
		"**/external/**/*.gzip",
		"**/external/**/.*ignore",

		"**/external/foobar/*.css",
		"**/external/barfoo/less",
		"**/external/barfoo/scss",
	}
	object := CompileIgnoreLines(lines...)
	assert.NotNil(test, object, "returned object should not be nil")

	assert.True(test, object.MatchesPath("external/foobar/angular.foo.css"), "external/foobar/angular.foo.css")
	assert.True(test, object.MatchesPath("external/barfoo/.gitignore"), "external/barfoo/.gitignore")
	assert.True(test, object.MatchesPath("external/barfoo/.bower.json"), "external/barfoo/.bower.json")
}

func TestCompileIgnoreLinesCarriageReturn(test *testing.T) {
	lines := []string{"abc/def\r", "a/b/c\r", "b\r"}
	object := CompileIgnoreLines(lines...)

	assert.True(test, object.MatchesPath("abc/def/child"), "abc/def/child should match")
	assert.True(test, object.MatchesPath("a/b/c/d"), "a/b/c/d should match")

	assert.False(test, object.MatchesPath("abc"), "abc should not match")
	assert.False(test, object.MatchesPath("def"), "def should not match")
	assert.False(test, object.MatchesPath("bd"), "bd should not match")
}

func TestCompileIgnoreLinesWindowsPath(test *testing.T) {
	if runtime.GOOS != "windows" {
		return
	}
	lines := []string{"abc/def", "a/b/c", "b"}
	object := CompileIgnoreLines(lines...)

	assert.True(test, object.MatchesPath("abc\\def\\child"), "abc\\def\\child should match")
	assert.True(test, object.MatchesPath("a\\b\\c\\d"), "a\\b\\c\\d should match")
}

func TestWildCardFiles(test *testing.T) {
	gitIgnore := []string{"*.swp", "/foo/*.wat", "bar/*.txt"}
	object := CompileIgnoreLines(gitIgnore...)

	// Paths which are targeted by the above "lines"
	assert.True(test, object.MatchesPath("yo.swp"), "should ignore all swp files")
	assert.True(test, object.MatchesPath("something/else/but/it/hasyo.swp"), "should ignore all swp files in other directories")

	assert.True(test, object.MatchesPath("foo/bar.wat"), "should ignore all wat files in foo - nonpreceding /")
	assert.True(test, object.MatchesPath("/foo/something.wat"), "should ignore all wat files in foo - preceding /")

	assert.True(test, object.MatchesPath("bar/something.txt"), "should ignore all txt files in bar - nonpreceding /")
	assert.True(test, object.MatchesPath("/bar/somethingelse.txt"), "should ignore all txt files in bar - preceding /")

	// Paths which are not targeted by the above "lines"
	assert.False(test, object.MatchesPath("something/not/infoo/wat.wat"), "wat files should only be ignored in foo")
	assert.False(test, object.MatchesPath("something/not/infoo/wat.txt"), "txt files should only be ignored in bar")
}

func TestPrecedingSlash(test *testing.T) {
	gitIgnore := []string{"/foo", "bar/"}
	object := CompileIgnoreLines(gitIgnore...)

	assert.True(test, object.MatchesPath("foo/bar.wat"), "should ignore all files in foo - nonpreceding /")
	assert.True(test, object.MatchesPath("/foo/something.txt"), "should ignore all files in foo - preceding /")

	assert.True(test, object.MatchesPath("bar/something.txt"), "should ignore all files in bar - nonpreceding /")
	assert.True(test, object.MatchesPath("/bar/somethingelse.go"), "should ignore all files in bar - preceding /")
	assert.True(test, object.MatchesPath("/boo/something/bar/boo.txt"), "should block all files if bar is a sub directory")

	assert.False(test, object.MatchesPath("something/foo/something.txt"), "should only ignore top level foo directories- not nested")
}

func BenchmarkCompileIgnoreLines(b *testing.B) {
	for i := 0; i < b.N; i++ {
		lines := []string{"abc/def", "a/b/c", "b"}
		object := CompileIgnoreLines(lines...)

		// MatchesPath
		// Paths which are targeted by the above "lines"
		assert.Equal(b, true, object.MatchesPath("abc/def/child"), "abc/def/child should match")
		assert.Equal(b, true, object.MatchesPath("a/b/c/d"), "a/b/c/d should match")

		// Paths which are not targeted by the above "lines"
		assert.Equal(b, false, object.MatchesPath("abc"), "abc should not match")
		assert.Equal(b, false, object.MatchesPath("def"), "def should not match")
		assert.Equal(b, false, object.MatchesPath("bd"), "bd should not match")

		object = CompileIgnoreLines("abc/def", "a/b/c", "b")

		// Paths which are targeted by the above "lines"
		assert.Equal(b, true, object.MatchesPath("abc/def/child"), "abc/def/child should match")
		assert.Equal(b, true, object.MatchesPath("a/b/c/d"), "a/b/c/d should match")

		// Paths which are not targeted by the above "lines"
		assert.Equal(b, false, object.MatchesPath("abc"), "abc should not match")
		assert.Equal(b, false, object.MatchesPath("def"), "def should not match")
		assert.Equal(b, false, object.MatchesPath("bd"), "bd should not match")
	}
}
