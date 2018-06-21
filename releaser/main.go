package main

/*
Minimal tool to automate release creation.

Create:
- git tag
- homebrew bottle
- linux tarball
- GitHub release with asset link(s)

Update:
- Homebrew formula tap with new release & SHAs

*/

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/google/go-github/github"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"golang.org/x/oauth2"
)

const (
	repoOwner = "dollarshaveclub"
	repoName  = "psst"
)

var rname, npath, commitsha, ghtoken, taprepo, tapref, fpath, ftpath, targetoslist string
var draft, prerelease, dobuild bool
var trowner, trname string
var hbrev, brbd uint
var osvs []string

var logger = log.New(os.Stderr, "", log.LstdFlags)

func ferr(msg string, args ...interface{}) {
	fmt.Printf(msg+"\n", args...)
	os.Exit(1)
}

var ghc *github.Client

func init() {
	pflag.StringVar(&rname, "release", "", "release name (ex: v1.0.0)")
	pflag.StringVar(&npath, "notes-path", "relnotes.md", "path to release notes")
	pflag.StringVar(&commitsha, "commit", "", "commit SHA to release")
	pflag.StringVar(&taprepo, "tap-repo", "dollarshaveclub/homebrew-public", "name of tap GitHub repository ([owner]/[repo])")
	pflag.StringVar(&tapref, "tap-repo-ref", "master", "tap repository ref (branch/tag/SHA)")
	pflag.StringVar(&fpath, "formula", "Formula/psst.rb", "path to formula within tap repo")
	pflag.StringVar(&ftpath, "formula-template", "Formula/psst.rb.tmpl", "path to formula template within tap repo")
	pflag.StringVar(&targetoslist, "macos-versions", "el_capitan,high_sierra,sierra", "Supported MacOS versions (comma-delimited)")
	pflag.UintVar(&hbrev, "homebrew-rev", 1, "Homebrew revision (bump to force reinstall/rebuild)")
	pflag.UintVar(&brbd, "bottle-rebuild", 1, "Bottle rebuild (bump to force bottle reinstall)")
	pflag.BoolVar(&draft, "draft", false, "Draft release (unpublished)")
	pflag.BoolVar(&prerelease, "prerelease", false, "Prerelease")
	pflag.BoolVar(&dobuild, "build", true, "Build binaries first")
	pflag.Parse()
	trs := strings.Split(taprepo, "/")
	if len(trs) != 2 {
		ferr("malformed tap repo (expected [owner]/[repo]): %v", taprepo)
	}
	if rname == "" {
		ferr("release name is required")
	}
	trowner = trs[0]
	trname = trs[1]
	osvs = strings.Split(targetoslist, ",")
	if len(osvs) == 0 {
		ferr("At least one MacOS version is required")
	}
	ghtoken = os.Getenv("GITHUB_TOKEN")
	if ghtoken == "" {
		ferr("GITHUB_TOKEN missing from environment")
	}
	if err := checkFiles(npath); err != nil {
		ferr("file path error: %v", err)
	}
	checkLocalRepoVersion()
	ghc = newGHClient()
}

func newGHClient() *github.Client {
	tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghtoken},
	))
	return github.NewClient(tc)
}

func checkLocalRepoVersion() {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		ferr("error getting git command output: %v", err)
	}
	if strings.TrimRight(string(out), "\n") != commitsha {
		ferr("current git revision does not match requested release version: %v (expected %v)", string(out), commitsha)
	}
}

func checkFiles(paths ...string) error {
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			return errors.Wrap(err, "file error")
		}
	}
	return nil
}

func createGitTag() error {
	msg := fmt.Sprintf("release %v", rname)
	ot := "commit"
	tag := github.Tag{
		Tag:     &rname,
		Message: &msg,
		Object: &github.GitObject{
			Type: &ot,
			SHA:  &commitsha,
		},
	}
	log.Printf("creating tag...\n")
	_, _, err := ghc.Git.CreateTag(context.Background(), repoOwner, repoName, &tag)
	if err != nil {
		return errors.Wrap(err, "error creating tag")
	}
	refstr := fmt.Sprintf("refs/tags/%v", rname)
	objt := "commit"
	ref := github.Reference{
		Ref: &refstr,
		Object: &github.GitObject{
			Type: &objt,
			SHA:  &commitsha,
		},
	}
	log.Printf("creating tag ref...\n")
	_, _, err = ghc.Git.CreateRef(context.Background(), repoOwner, repoName, &ref)
	if err != nil {
		return errors.Wrap(err, "error creating tag ref")
	}
	return nil
}

type bottleDefinition struct {
	Hash     string
	TargetOS string
}

type formulaTemplateData struct {
	Tag              string
	CommitSHA        string
	HomebrewRevision uint
	BaseDownloadURL  string
	Bottled          bool
	BottleRebuild    uint
	BottleDefs       []bottleDefinition
}

func (ftd *formulaTemplateData) populate(bdefs []bottleDefinition) {
	ftd.Tag = rname
	ftd.CommitSHA = commitsha
	ftd.HomebrewRevision = hbrev
	ftd.BaseDownloadURL = fmt.Sprintf("https://github.com/%v/%v/releases/download/%v/", repoOwner, repoName, rname)
	ftd.BottleRebuild = brbd
	ftd.Bottled = true
	ftd.BottleDefs = bdefs
}

const header = "# GENERATED FROM TEMPLATE. DO NOT EDIT!\n"

// generateFormula fetches the template from github, executes the template with ftd and returns the raw data or error, if any
func generateFormula(ftd formulaTemplateData) ([]byte, error) {
	logger.Printf("Generating Homebrew formula")
	// get template
	fc, _, _, err := ghc.Repositories.GetContents(context.Background(), trowner, trname, ftpath, &github.RepositoryContentGetOptions{Ref: tapref})
	if err != nil {
		return nil, errors.Wrap(err, "error getting formula template")
	}
	rt, err := fc.GetContent()
	if err != nil {
		return nil, errors.Wrap(err, "error getting formula template content")
	}
	// generate new formula
	tmpl, err := template.New("formula").Parse(rt)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing formula template")
	}
	buf := bytes.NewBuffer([]byte{})
	if err = tmpl.Execute(buf, &ftd); err != nil {
		return nil, errors.Wrap(err, "error executing template")
	}
	return append([]byte(header), buf.Bytes()...), nil
}

func pushFormula(fd []byte) error {
	logger.Printf("Pushing Homebrew formula")
	// Get the current file for the SHA
	fc, _, _, err := ghc.Repositories.GetContents(context.Background(), trowner, trname, fpath, &github.RepositoryContentGetOptions{Ref: tapref})
	if err != nil {
		return errors.Wrap(err, "error getting formula contents")
	}
	sp := func(s string) *string {
		return &s
	}
	_, _, err = ghc.Repositories.UpdateFile(context.Background(), trowner, trname, fpath, &github.RepositoryContentFileOptions{
		Message: sp(fmt.Sprintf("updated for release %v", rname)),
		Content: fd,
		SHA:     fc.SHA,
		Branch:  &tapref,
	})
	if err != nil {
		return errors.Wrap(err, "error updating formula")
	}
	return nil
}

const (
	linuxBinName = "psst-linux-amd64"
)

var buildopts = []string{"-ldflags", "-X github.com/dollarshaveclub/psst/internal/version.CommitSHA=%v -X github.com/dollarshaveclub/psst/internal/version.Version=%v -X $(REPO)/cmd.CompiledDirectory=github -X $(REPO)/cmd.CompiledStorage=vault -X $(REPO)/cmd.Org=dollarshaveclub"}

func buildBins() error {
	if err := os.MkdirAll("bins", os.ModeDir|0755); err != nil {
		return errors.Wrap(err, "error creating bins directory")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "error getting working directory")
	}
	wd := filepath.Join(cwd, "..")
	buildopts[1] = fmt.Sprintf(buildopts[1], commitsha, rname)
	build := func(osn string) ([]byte, error) {
		cmd := exec.Command("go", append([]string{"build"}, buildopts...)...)
		cmd.Env = append(os.Environ(), []string{fmt.Sprintf("GOOS=%v", osn), "GOARCH=amd64"}...)
		cmd.Dir = wd
		return cmd.CombinedOutput()
	}
	logger.Printf("Building binaries...\n")
	logger.Printf("...macOS amd64")
	if out, err := build("darwin"); err != nil {
		return errors.Wrapf(err, "error running build command: %s", out)
	}
	if err := os.Rename(filepath.Join(wd, "psst"), filepath.Join(cwd, "bins", "psst-darwin")); err != nil {
		return errors.Wrap(err, "error renaming binary")
	}
	logger.Printf("...Linux amd64")
	if out, err := build("linux"); err != nil {
		return errors.Wrapf(err, "error running build command: %s", out)
	}
	lfn := filepath.Join(cwd, "bins", linuxBinName)
	if err := os.Rename(filepath.Join(wd, "psst"), lfn); err != nil {
		return errors.Wrap(err, "error renaming binary")
	}
	// compress linux binary
	logger.Printf("...compressing Linux binary\n")
	d, err := ioutil.ReadFile(lfn)
	if err != nil {
		return errors.Wrap(err, "error reading linux binary")
	}
	f, err := os.Create(lfn + ".gz")
	if err != nil {
		return errors.Wrap(err, "error creating compressed linux binary")
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	defer gw.Close()
	if _, err := gw.Write(d); err != nil {
		return errors.Wrap(err, "error writing compressed linux binary")
	}
	return nil
}

// "copy" (link) a file if it doesn't exist
func cpifneeded(src, dest string) error {
	if _, err := os.Stat(dest); err != nil {
		if os.IsNotExist(err) {
			return os.Link(src, dest)
		}
		return errors.Wrap(err, "error getting destination")
	}
	return nil
}

var bottleNameTmpl = template.Must(template.New("bn").Parse("psst-{{ .Release }}_{{ .HomebrewRevision }}.{{ .OS }}.bottle.{{ .BottleRebuild }}.tar.gz"))

// createBottle synthetically creates a bottle tarball returning the bottle definitions, local bottle filenames and error if any
func createBottle() ([]bottleDefinition, []string, error) {
	logger.Printf("Creating Homebrew bottle...\n")
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, errors.Wrap(err, "error getting working directory")
	}
	basepath := filepath.Join(".", "psst", rname)
	binpath := filepath.Join(basepath, "bin")
	if err := os.MkdirAll(binpath, os.ModeDir|0755); err != nil {
		return nil, nil, errors.Wrap(err, "error creating bottle directory path")
	}
	// .brew
	if err := os.MkdirAll(filepath.Join(basepath, ".brew"), os.ModeDir|0755); err != nil {
		return nil, nil, errors.Wrap(err, "error creating .brew directory")
	}
	// copy README
	if err := cpifneeded(filepath.Join(cwd, "..", "README.md"), filepath.Join(basepath, "README.md")); err != nil {
		return nil, nil, errors.Wrap(err, "error copying README")
	}
	// copy binary
	if err := cpifneeded(filepath.Join("bins", "psst-darwin"), filepath.Join(binpath, "psst")); err != nil {
		return nil, nil, errors.Wrap(err, "error copying binary")
	}
	// INSTALL_RECEIPT.json
	ir, err := ioutil.ReadFile("INSTALL_RECEIPT.json.tmpl")
	if err != nil {
		return nil, nil, errors.Wrap(err, "error reading install receipt template")
	}
	tmpl, err := template.New("instrcpt").Parse(string(ir))
	d := struct {
		Release          string
		OS               string
		HomebrewRevision uint
		BottleRebuild    uint
	}{
		Release:          rname,
		HomebrewRevision: hbrev,
		BottleRebuild:    brbd,
	}
	buf := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(buf, &d); err != nil {
		return nil, nil, errors.Wrap(err, "error executing install receipt template")
	}
	if err := ioutil.WriteFile(filepath.Join(basepath, "INSTALL_RECEIPT.json"), buf.Bytes(), os.ModePerm); err != nil {
		return nil, nil, errors.Wrap(err, "error writing install receipt")
	}
	// tar it up
	if err := os.MkdirAll("bottle", os.ModeDir|0755); err != nil {
		return nil, nil, errors.Wrap(err, "error creating bottle directory")
	}
	buf = bytes.NewBuffer([]byte{})
	d.OS = osvs[0]
	if err := bottleNameTmpl.Execute(buf, &d); err != nil {
		return nil, nil, errors.Wrap(err, "error executing bottle filename template: "+d.OS)
	}
	bp := filepath.Join("bottle", buf.String())
	if err := archiver.TarGz.Make(bp, []string{"psst"}); err != nil {
		return nil, nil, errors.Wrap(err, "error creating bottle tarball")
	}
	// Get hash of bottle, populate bottle definitions
	bd, err := ioutil.ReadFile(bp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error reading bottle")
	}
	sha := fmt.Sprintf("%x", sha256.Sum256(bd))
	bdefs := []bottleDefinition{
		bottleDefinition{
			Hash:     sha,
			TargetOS: osvs[0],
		},
	}
	lps := []string{bp}
	// link other bottles
	for _, osn := range osvs[1:] {
		d.OS = osn
		buf = bytes.NewBuffer([]byte{})
		if err := bottleNameTmpl.Execute(buf, &d); err != nil {
			return nil, nil, errors.Wrap(err, "error executing bottle filename template: "+d.OS)
		}
		p := filepath.Join("bottle", buf.String())
		if err := cpifneeded(bp, p); err != nil {
			return nil, nil, errors.Wrap(err, "error linking bottle")
		}
		lps = append(lps, p)
		bdefs = append(bdefs, bottleDefinition{
			Hash:     sha,
			TargetOS: osn,
		})
	}

	return bdefs, lps, nil
}

func createGHRelease(assetpaths []string) error {
	rel := github.RepositoryRelease{
		TagName: &rname,
		//TargetCommitish: &commitsha,
		Name:       &rname,
		Draft:      &draft,
		Prerelease: &prerelease,
	}
	nd, err := ioutil.ReadFile(npath)
	if err != nil {
		return errors.Wrap(err, "error reading release notes")
	}
	notes := string(nd)
	rel.Body = &notes
	logger.Printf("Creating GitHub release")
	ro, _, err := ghc.Repositories.CreateRelease(context.Background(), repoOwner, repoName, &rel)
	if err != nil {
		return errors.Wrap(err, "error creating release")
	}
	for _, ap := range assetpaths {
		f, err := os.Open(ap)
		if err != nil {
			return errors.Wrap(err, "error opening asset")
		}
		defer f.Close()
		logger.Printf("Uploading asset %v...", ap)
		resp, _, err := ghc.Repositories.UploadReleaseAsset(context.Background(), repoOwner, repoName, *ro.ID, &github.UploadOptions{Name: filepath.Base(ap)}, f)
		if err != nil {
			return errors.Wrap(err, "error uploading asset")
		}
		logger.Printf("...%v\n", resp.GetBrowserDownloadURL())
	}
	return nil
}

func cleanup() error {
	logger.Printf("Cleaning up")
	for _, p := range []string{"./bins", "./bottle", "./psst"} {
		if err := os.RemoveAll(p); err != nil {
			return errors.Wrap(err, "error removing path")
		}
	}
	return nil
}

func main() {
	if dobuild {
		if err := buildBins(); err != nil {
			ferr("error building binaries: %v", err)
		}
	}
	bds, lps, err := createBottle()
	if err != nil {
		ferr("error creating bottle: %v", err)
	}
	ftd := formulaTemplateData{}
	ftd.populate(bds)
	fd, err := generateFormula(ftd)
	if err != nil {
		ferr("error generating formula: %v", err)
	}
	if err = pushFormula(fd); err != nil {
		ferr("error pushing formula: %v", err)
	}
	if err := createGitTag(); err != nil {
		ferr("error creating tag: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		ferr("error getting working directory: %v", err)
	}
	assetpaths := append([]string{filepath.Join(cwd, "bins", linuxBinName+".gz")}, lps...)
	if err = createGHRelease(assetpaths); err != nil {
		ferr("error creating GitHub release: %v", err)
	}
	if err := cleanup(); err != nil {
		ferr("error cleaning up: %v", err)
	}
	logger.Printf("Done")
}
