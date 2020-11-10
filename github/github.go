package github

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/actions-go/toolkit/core"
	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

func token() string {
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t
	}
	for _, input := range []string{"github-token", "token"} {
		if t, ok := core.GetInput(input); ok {
			return t
		}
	}
	return ""
}

func githubHTTPClient(client *http.Client) *http.Client {
	token := token()
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		return oauth2.NewClient(context.Background(), ts)
	}
	return http.DefaultClient

}

func NewClient() *github.Client {
	c := github.NewClient(githubHTTPClient(nil))
	u, err := url.Parse(APIURL())
	if err == nil {
		if !strings.HasSuffix(u.Path, "/") {
			u.Path = u.Path + "/"
		}
		c.BaseURL = u
	}
	return c
}

var GitHub = NewClient()

func authorize(r *http.Request) {
	t := token()
	if t != "" {
		r.SetBasicAuth("", t)
	}
}

func readTarResponse(resp *http.Response, stripFolder int, include Matcher) (map[string]RepositoryFile, error) {
	var body io.Reader = resp.Body
	var err error
	switch resp.Header.Get("Content-Type") {
	case "application/gzip", "application/x-gzip":
		body, err = gzip.NewReader(body)
		if err != nil {
			return nil, err
		}
	case "application/zip":
		b := bytes.NewBuffer(nil)
		written, err := io.Copy(b, resp.Body)
		fmt.Println(written)
		if err != nil {
			return nil, err
		}

		r, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
		if err != nil {
			return nil, err
		}
		files := map[string]RepositoryFile{}
		for _, f := range r.File {
			if !f.FileInfo().IsDir() {
				if include(f.Name) {
					core.Debugf("Downloading %v", f.Name)
					rd, err := f.Open()
					if err != nil {
						return nil, err
					}
					b, err := ioutil.ReadAll(rd)
					if err != nil {
						return nil, err
					}
					files[f.Name] = RepositoryFile{
						Path:     f.Name,
						FileInfo: f.FileInfo(),
						Data:     b,
					}
				}
			}
		}
		return files, nil
	}
	files := map[string]RepositoryFile{}
	tr := tar.NewReader(body)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, err
		}
		if hdr.Format == tar.FormatPAX || hdr.FileInfo().IsDir() {
			continue
		}
		name := hdr.Name
		if stripFolder > 0 {
			l := strings.SplitN(hdr.Name, string(os.PathSeparator), stripFolder+1)
			if len(l) <= stripFolder {
				core.Warningf("skipping %s from tarball, it is in below the stripped folder level %d", hdr.Name, stripFolder)
				continue
			}
			name = l[stripFolder]
		}

		if include(name) {
			core.Debugf("Downloading %v", hdr.Name)
			b := bytes.NewBuffer(nil)
			if _, err := io.Copy(b, tr); err != nil {
				return nil, err
			}
			files[name] = RepositoryFile{
				Path:     name,
				FileInfo: hdr.FileInfo(),
				Data:     b.Bytes(),
			}
		}
	}
	return files, nil
}

type Matcher func(path string) bool

type RepositoryFile struct {
	Path     string
	FileInfo os.FileInfo
	Data     []byte
}

// DownloadSelectedRepositoryFiles downloads files from a given repository and branch, given that their name matches regarding the `include` function
func DownloadSelectedRepositoryFiles(c *http.Client, owner, repo, branch string, include Matcher) map[string]RepositoryFile {
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/tarball/%s", owner, repo, branch)
	core.Debugf("Downloading tarball for repo: %s", u)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		core.Warningf("failed to download repository: %v", err)
		return nil
	}
	authorize(req)
	resp, err := c.Do(req)
	if err != nil {
		core.Warningf("failed to download repository: %v", err)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		core.Warningf("failed to download repository: unexpected code %d", resp.StatusCode)
		return nil
	}
	defer resp.Body.Close()
	r, err := readTarResponse(resp, 1, include)
	if err != nil {
		core.Warningf("failed to download repository: %v", err)
		return nil
	}
	return r
}

// MatchesOneOf returns a matcher returning whether the path matches one of the provided glob patterns
func MatchesOneOf(patterns ...string) Matcher {
	return func(path string) bool {
		for _, p := range patterns {
			exp, err := regexp.CompilePOSIX(p)
			if err != nil {
				core.Warningf("unable to compile pattern %s: %v", p, err)
			}
			if exp.MatchString(path) {
				return true
			}
		}
		return false
	}
}

// MatchAll implements a Matcher that matches any name
func MatchAll(string) bool {
	return true
}

// DownloadArtifact downloads a workflow artifact by its name
func DownloadArtifact(name string) (map[string]RepositoryFile, error) {
	repo := strings.SplitN(Repository(), "/", 2)
	artifacts, _, err := GitHub.Actions.ListWorkflowRunArtifacts(context.Background(), getIndex(repo, 0), getIndex(repo, 1), int64(RunID()), &github.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, artifact := range artifacts.Artifacts {
		if artifact.GetName() == name {
			u, _, err := GitHub.Actions.DownloadArtifact(context.Background(), getIndex(repo, 0), getIndex(repo, 1), *artifact.ID, true)
			if err != nil {
				return nil, err
			}
			r, err := http.NewRequest("GET", u.String(), nil)
			if err != nil {
				return nil, err
			}
			resp, err := githubHTTPClient(nil).Do(r)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			return readTarResponse(resp, 0, MatchAll)
		}
	}
	return nil, fmt.Errorf("unable to find artfifact named %s for run %d on repository %s", name, RunID(), Repository())
}
